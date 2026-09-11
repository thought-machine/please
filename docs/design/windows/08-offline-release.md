# The Offline Windows Release

Status: **Implemented** · Milestone: M4a · Last updated: 2026-09-11

How to build a `windows_amd64` zip that works on a machine with no network and no
configuration: plugins resolved from inside the install, helper tools beside the binary.

This is a **workaround for internal use**, and it exists only because the Windows fixes for all
four language plugins sit on unpublished branches. It may be long-lived — there is no push
access to any of the plugin repos and no date for one — so it is designed to be maintained
rather than to be temporary. `07-state-of-play.md` carries the removal checklist for whoever
eventually publishes those branches.

## What is wrong with today's zip

`//package:release_files` already produces `please_<VERSION>.zip` holding `please.exe`,
`busybox.exe`, `build_langserver.exe` and `plz.cmd`. Extracted on a real Windows machine it
cannot build anything.

- **The plugins are downloaded at parse time.** `plugins/BUILD` calls `plugin_repo()`, which
  fetches a GitHub archive at a pinned tag. Those tags do not contain any of the Windows work,
  so the user gets plugins that cannot build for Windows — and needs network access to get even
  those.
- **The helper tools have no Windows release.** `arcat` gates parsing the moment any plugin is
  involved. `please_go`, `please_cc` and `please_pex` gate their languages.

## Two rules this design obeys

**Nothing binary goes into git.** The plugin archives and the zip are generated artifacts. The
archives are produced by a script into a gitignored directory and consumed as ordinary sources,
so Please hashes their contents the way it hashes anything else.

**The normal release path does not change.** CI has no plugin checkouts and never will, so the
`build-windows` job keeps producing exactly the artifact it produces today. The bundled zip is
built by whoever has the four checkouts, which is the same precondition `.plzconfig.local`
already imposes on anyone working on this port. One `[buildconfig]` key turns the bundling on.

## How resolution works

Three mechanisms, all of which already exist. Nothing new is invented.

1. **`remote_file` tries each URL in turn and stops at the first success** (`fetchRemoteFile`,
   `src/build/build_step.go`), and it understands `file://` URLs whose path is absolute and
   outside the repo root. Prepending one template to `Please.PluginRepo` therefore gives
   bundled-first with GitHub fallback, with no change to `plugin_repo()` at all.
2. **`config.Please.Location` is always the head of the build action PATH** (`getBuildEnv`,
   `src/core/config.go`), and a bare tool name resolves through `core.LookPath`, which appends
   `.exe` on Windows. That is how the bundled `busybox` is found today; the bundled tools ride
   the same route. No path is constructed anywhere.
3. **Anything added to `//package:installed_files` lands flat under `please/` in the zip**,
   because `//package:please_zip` runs `arcat zip --dumb --input package --rename_dir
   package:please`. The zip rule, `release_files` and the CI job need no changes.

### Why the payload must be flat files

`pleasew.ps1` extracts into `<Location>/<version>/` and links the contents back up a level with
`Get-ChildItem -File` — **files only**. The self-updater's `linkNewPlease` does the same thing
with `os.ReadDir` and `linkFile`. Meanwhile `EnsurePleaseLocation()` forces `Location` to
exactly `~/.please` for any executable underneath it.

So a `plugins/` subdirectory would be stranded at `~/.please/<version>/plugins` while Please
looked for it at `~/.please`. Flat files at the top of `please/` get linked up and are found.

## Where the plugin archives come from

A script, `tools/misc/vendor_plugins.sh`, writes them into `third_party/plugins/`, which is
gitignored.

For each checkout it refuses unless the tree is clean and on branch `windows`, then runs
`git archive --format=zip --prefix="<name>-<sha>/" HEAD`. Three reasons for `git archive`
rather than the working tree:

- The working trees carry `plz-out/`, `.plzconfig.local` and whatever else is untracked.
- The commit SHA is the only durable identifier these branches have.
- It is deterministic for a fixed commit, and none of the four checkouts has an `export-ignore`.

The `--prefix` produces exactly one top-level directory containing a `.plzconfig`, which is
what `plugin_repo()`'s extract step requires.

The script also writes `plugin_revisions.txt`, which ships inside the zip, so a bug report from
a Windows user carries its own provenance.

### Why not a build rule

Every route was considered and none works.

| Route | Why not |
|---|---|
| `genrule` reading `CONFIG.GO_RULES_PATH` | The checkout path appears only in `cmd`, so the rule hash covers the path string rather than the tree. An edited plugin would be served stale from cache forever. |
| `remote_file` with a `file://` URL | Copies one file. It cannot zip a tree, so the zip would have to exist already. |
| Depending on a target in the subrepo | A local subrepo is `os.DirFS(root)` with no build target, so nothing in the graph can depend on "its files". Adding an `all_srcs` filegroup to each plugin would make bundling depend on a patch to the thing being bundled. |

A script writing real files into the repo, consumed by an ordinary `filegroup`, hashes
correctly and couples to nothing. It is the same shape as `plz puku sync`: an out-of-band step
with a committed result, except that here the result is gitignored rather than committed.

### The filenames carry no revision

`plugin_go-rules.zip`, not `plugin_go-rules_v1.31.1.zip`, and the `file://` template is
`file://<Location>/plugin_{plugin}.zip`.

Putting the revision in would mean offline resolution only works for a repo pinning exactly the
bundled revision, and `plz init plugin` writes whatever the latest upstream tag is, so a real
repo would miss and fall through to the network — defeating the point.

The cost is that a repo pinning a *different* version of a plugin silently gets ours on
Windows. `plugin_revisions.txt` has to say so in words, and so do the release notes. It must
also say that these are not the upstream releases they are versioned against: they are
`v1.31.1+2`, `v0.7.3+2`, `v0.2.1+3` and `v2.0.2+1`, each that tag plus unmerged Windows
commits. Deleting an archive from the install opts back out.

## arcat needs no clone and no blob

It can be built here from the module proxy. Of arcat v1.3.1's requirements, all but two are
already in `third_party/go/BUILD`; `github.com/please-build/ar` and `github.com/xi2/xz` are
missing. Three `go_repo()` entries, and the binary target is
`///third_party/go/github.com_please-build_arcat//:arcat`. That pattern is already in use:
`docs/build_defs/docs.build_defs` takes `claat` exactly that way.

**Measured, and it works.** The target builds on Linux and cross-builds to a PE32+ binary
under `--arch windows_amd64`, and the older `klauspost/compress` it asks for is satisfied by
the version already pinned. The `go 1.17` directive in arcat's `go.mod`, which
`04-release-and-ci.md` flags as a blocker, never comes up: `please_go` invokes `go tool compile`
per package and never passes `-lang`.

**But arcat had two Windows bugs, and either one stops it writing a zip at all.** Both are
carried as `third_party/go/arcat_windows_rename.patch`, applied through `go_repo`'s `patch`
argument, because there is nowhere to push them upstream yet.

1. The output file is created with `ioutil.TempFile`, which returns it open, and that handle is
   never closed — `zip.NewFile` opens the same path again and closes only its own. Unix does not
   care that a file being renamed is still open; Windows fails with `Sharing violation`. This is
   the `ERROR_SHARING_VIOLATION` class that `05-testing-strategy.md` names as the likeliest
   source of real-Windows-only failures, and it turned up on the first thing that was tried.
2. `filepath.WalkDir` hands back OS-separated paths and they went straight into zip member
   names. A zip member name is always `/`-separated, so on Windows every name carried
   backslashes, `--rename_dir` and `--strip_prefix` silently matched nothing, and no reader
   split the names into directories. The `filepath`-for-`path` trap again, in a third language
   after Go and Python.

`please_go`, `please_cc` and `please_pex` have source in the plugin checkouts and build from
them directly — but **not from here**. Referencing `///go//tools/please_go:please_go` and
friends under a Windows arch collides on subrepo names: the plugin's `third_party/go` and ours
both register e.g. `third_party/go/github.com_stretchr_testify@windows_amd64`, because an arch
subrepo's name does not carry the subrepo that owns it. That is a bug in Please, and a deep one
— the fix changes every subrepo name and so every hash. `vendor_plugins.sh` builds them inside
their own repos instead, where there is nothing to collide with.

## Changes

### 1. `third_party/go/BUILD`

`go_repo` entries for `github.com/xi2/xz`, `github.com/please-build/ar` and
`github.com/please-build/arcat` at v1.3.1. Confirm both build directions before going on.

### 2. `tools/misc/vendor_plugins.sh` and `third_party/plugins/`

The script as described above. `third_party/plugins/BUILD` is committed and lists the four
zips explicitly — explicitly rather than by `glob()`, so that a missing archive is an error
rather than a zip that silently ships without plugins. The whole package is guarded by
`CONFIG.get("BUNDLED_PLUGINS")` so that an ordinary build with no vendored archives parses
cleanly. `.gitignore` gains `/third_party/plugins/*.zip` and `/third_party/plugins/plugin_revisions.txt`.

### 3. `src/core/config.go`

This is the one piece that is not a workaround. "A Windows install can carry its plugins beside
it" is defensible on its own terms and should survive the plugin branches being published.

- Hoist the arcat default to a `DefaultArcatTool` constant and use it in
  `src/parse/internal_package.go`'s `ArcatUnavailable`, which currently rebuilds the same
  literal.
- Move the `EnsurePleaseLocation()` call to before the `setDefault` block. It is idempotent and
  reads only already-populated state, so the move is safe.
- Replace the inline `setDefault(&config.Please.PluginRepo, ...)` list with a
  `defaultPluginRepos()` method that prepends `file://<Location>/plugin_{plugin}.zip` when
  `runtime.GOOS == "windows"`. Run the location through `filepath.ToSlash`.
- Add `useBundledTools()`, called just after, setting `Build.ArcatTool` to the bare name
  `"arcat"` when the platform is Windows, the tool is still the default, and
  `<Location>/arcat.exe` exists. The existence check earns its keep: without it a Windows user
  with no bundle gets a `SystemPathLabel` that panics in `FullPaths` rather than the civil
  warning they get today.

Gating on `runtime.GOOS` rather than on file existence or the target arch is what keeps Linux
and macOS provably untouched. The URL list is hashed into every plugin download's rule hash, so
an extra template would change hashes everywhere. It also leaves our own cross-build alone:
`ForArch` copies the host config and never re-reads it, so `plz build --arch windows_amd64` on
Linux still fetches plugins exactly as it does now.

Tests in `src/core/config_test.go`: `defaultPluginRepos` returns exactly two entries off
Windows, and `useBundledTools` is a no-op when the tool was set explicitly.

### 4. The four plugin branches

Each plugin's helper tool must default to the bundled binary on a Windows host. The pattern is
already established here — shell-rules moved its shell default out of
`.plzconfig_windows_amd64`, which is never read when a repo is used as a plugin, and into a
per-call function.

| Repo | `.plzconfig` | Build defs |
|---|---|---|
| go-rules | `please_go_tool`: drop `DefaultValue`, add `Optional = true` | new `_please_go_tool()`, ten call sites |
| cc-rules | `please_cc_tool` likewise | new `_please_cc_tool()`, two call sites |
| python-rules | `pex_tool` likewise | new `_pex_tool()`, two call sites |

Two things are easy to get wrong. Key the default on `CONFIG.HOSTOS`, not `CONFIG.OS`: these
tools run on the machine doing the building, so a Linux host cross-compiling to Windows still
wants the Linux one. And the non-Windows fallback must be fully qualified
(`///go//tools:please_go`), because a value returned from a build def is resolved in the
caller's package, unlike a `DefaultValue` in `.plzconfig`.

Also widen `//tools/please_cc:please_cc` in cc-rules to `PUBLIC`; it is currently visible only
within that repo. Leave the `PexTool` override in `.plzconfig_windows_amd64` alone — this repo
cross-builds from Linux, so the new default would pick the released Linux `please_pex`, which
has no Windows preamble. Update its comment to say why it is still needed.

Re-run `vendor_plugins.sh` after committing these.

### 5. `package/BUILD`

On Windows only, a `genrule` copying arcat to `arcat.exe`. The rename is needed because the go
plugin names a binary after its rule with no extension, and Windows will not run a file whose
name has no `PATHEXT` extension — the same reason `//src:please` asks for `please.exe`.

Then, in the Windows branch of `installed_files`, add `:arcat` and, when
`CONFIG.get("BUNDLED_PLUGINS")` is set, `//third_party/plugins:bundled`, which carries the four
archives, the three plugin tools and `plugin_revisions.txt`.

arcat goes in unconditionally; it is built from source here and needs no checkouts. Only the
plugin payload is gated.

## Verification

Be honest about what is provable. The zip bundles plugins and Please's own helper tools. It
does not bundle language toolchains and should not: cc needs a Windows-hosted MinGW, go needs a
Windows Go distribution whose hash is not in `third_party/go/BUILD` yet, and neither is on this
machine.

**Shape test, on Linux, no Wine.** A `gentest` that unzips `//package:please_zip`, asserts the
member list is exactly the expected set, and asserts each `plugin_*.zip` has one top-level
directory containing a `.plzconfig`. Cheap, and it catches the rename-to-`.exe` regressions
that would otherwise surface only on real Windows. Do it early; it gates the rest.

**The load-bearing test.** A `test/windows/offline_repo/` fixture modelled on `smoke_repo`,
with stock `plugin_repo()` calls for all four plugins preloaded and an `sh_binary` to build. A
`wine_plz_release_test` macro in `test/build_defs/wine.build_defs`, sibling to `wine_plz_test`,
which extracts the real `//package:please_zip` rather than assembling an install by hand. That
is the point: it tests the artifact, not a reconstruction of it.

Deny the network two ways, preferring the first: `unshare -rn` around the `wine` call (verify
unprivileged user namespaces work here and in the CI image), falling back to
`HTTP_PROXY=http://127.0.0.1:1` and friends, which every fetch dies on because the client uses
`ProxyFromEnvironment`.

Building that one `sh_binary` exercises the whole chain at once: the `file://` template
consulted four times, the bundled `arcat.exe` extracting four archives, all four plugins' build
defs parsing, shell-rules' bundled-busybox default, and busybox running the action.

**Add the negative control.** Without it the test proves nothing, since a warm cache or a stray
`~/.please` would pass it. Same test with one plugin archive deleted from the extracted install,
expecting failure.

That is `//test/windows:offline_release_test`, and the control is
`//test/windows:offline_release_negative_test`. Both are gated on `BUNDLED_PLUGINS` like the
bundling itself, so they are skipped in CI rather than failing there.

**The namespace half of the network denial is not available.** Wine aborts outright inside a
user namespace — `free(): invalid pointer` before it starts — so `unshare -rn` is out, and the
denial is a proxy pointed at a closed port. That proves no HTTP egress rather than no egress at
all, which is the right scope here since fetching a plugin is an HTTP fetch. The negative
control is what makes the pair rigorous.

**Still to do: a python tier.** Same harness plus the embeddable Python on `WINEPATH` as
`wine_pex_test` does, building and running a `python_binary` with the network denied. That is
the one that would exercise the bundled `please_pex.exe`, which nothing does yet.

Finally, `plz hash //...` on Linux before and after, to confirm nothing moved on the platforms
that already work, and the full three-pass `./test.sh`.

## Building one

```bash
rm -rf plz-out/pkg/windows_amd64          # see below
tools/misc/vendor_plugins.sh
plz build --arch windows_amd64 //package:release_files
# plz-out/pkg/windows_amd64/please_<VERSION>.zip
```

with `bundled-plugins = true` under `[buildconfig]` in `.plzconfig.local`, alongside the four
`*-rules-path` keys that are already there.

**`plz-out/pkg` does not update.** The `hlink:` label goes through `fs.LinkIfNotExists`, which
does nothing when the destination is already there, and the destination is named after the
version. So rebuilding a release at the same version leaves `plz-out/pkg` holding the previous
bytes, silently. `plz-out/gen/<arch>/package/` always has the real thing. This is pre-existing
and affects every platform; it cost an hour here, twice.

## Risks

- **`plz update` does not refresh the bundle.** The updater downloads a bare `please_<v>`
  binary, so after a self-update the new version directory holds only `please.exe` while the
  links at `~/.please` still point at the previous version's arcat and plugin zips. It keeps
  working, staler each time, silently. Pre-existing and out of scope here, but this design
  makes it load-bearing. Recorded in `07-state-of-play.md`.
- **A repo pinning a different plugin revision silently gets ours on Windows.** Accepted,
  documented, reversible by deleting the archive from the install.
- **A stale vendored archive.** Nothing forces `vendor_plugins.sh` to be re-run after a plugin
  commit. `plugin_revisions.txt` makes it visible in the artifact rather than preventing it.
- **Running an `sh_binary` in place complains.** Its payload unpacks beside it, which under
  `plz run` is `plz-out/bin/<pkg>/`, where the dependencies it is unpacking already sit as
  read-only build outputs. busybox reports `Permission denied` per file and carries on with
  what is already there, which happens to be identical. Pre-existing on Unix too, where it is
  silent because replacing a read-only file is allowed. Only affects running in place.
- **Version skew in arcat's module graph.** Only shows up at build time, in step 1.
- **`Optional` alongside `Inherit`** on `please_go_tool` and `pex_tool` is untested by this port
  so far; shell-rules' `shell_tool` was not inherited.
- **`unshare -rn` may be unavailable in the CI image**, leaving the weaker proxy denial.

## Documentation to update alongside

`04-release-and-ci.md` needs its "arcat — the real gate" section rewritten, since arcat stops
being a gate anywhere once it is built from the module proxy. `07-state-of-play.md` gets the
updater risk above and a removal checklist entry for the whole of this document's machinery,
for whoever publishes the plugin branches.
