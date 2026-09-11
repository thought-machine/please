# Milestone Tracker

Status: **Living document** · Last updated: 2026-09-10

> **M0 is done and it changed the plan.** `plz.exe` compiles, links, parses BUILD files and
> executes build actions under Wine after ~290 lines of probe changes. Estimates below are
> revised down accordingly. See `appendix-baseline-errors.md` for evidence.

The one file in this directory expected to change weekly. Update `Status` and `Notes` as work
lands; keep the exit criteria fixed unless the design genuinely changes, in which case update
the corresponding design doc too.

Legend: ⬜ not started · 🟡 in progress · ✅ done · ⚠️ blocked

## Summary

| # | Milestone | Est. | Status | Owner | Issue |
|---|---|---|---|---|---|
| M0 | Baseline and guardrail | 2d | ✅ | — | — |
| M1 | OS abstraction layer | 1–2w | ✅ | — | — |
| M2 | Paths, environment and the `.exe` model | 1w | ✅ | — | — |
| M3 | Build actions and the bundled shell | 1w | ✅ | — | — |
| M4 | Release pipeline: cross-built Windows artifacts | 1w | 🟡 | — | — |
| M5 | C++ on Windows: cc-rules (workstream B) | 2w | 🟡 | — | — |
| M6 | Linux-hosted verification harness | 1w | ✅ | — | — |
| M7 | Sandboxing parity | 2w | 🟡 | — | — |
| M8 | Remote execution and plugin parity | 3w | 🟡 | — | — |
| M9 | Native Windows CI and GA | 2w | ⬜ | — | — |

Rough total: 14–15 weeks of focused work. M0–M6 (the C++ vertical slice) is 7–8 weeks.

**M1 was re-estimated from 2–3 weeks to 1–2 weeks.** Making it *compile* turned out to be a
two-day job (5 sites). The remaining time is the part the compiler cannot help with: Job
Objects, `ExecReplace`, and real file locking — all silent runtime failures today.

## Sequencing

```
M0 ─┬─ M1 ─┬─ M2 ─┬─ M3 ─── M4 ─┐
    │      │      │             ├─ M6 ─┬─ M7 ─┬─ M9
    └─ M5 ─┴──────┴─────────────┘      └─ M8 ─┘
       (workstream B, parallel)
```

- **M5 can start immediately** and run in parallel with M1–M4. It is a different repo and its
  Loop B verification (`05-testing-strategy.md`) needs only a Linux `plz` and MinGW — neither
  of which depends on the core port.
- **M6 should start as soon as M1 produces a binary**, not after M5 finishes. The milestone
  number is a completion point.
- **M7 is not a blocker for anything.** `sandbox_other.go` already compiles on Windows and
  degrades to a plain `exec.Command`.

## M0 — Baseline and guardrail ✅

**Exit:** the non-blocking CI job runs, fails, and its output is recorded in
`appendix-baseline-errors.md`. — *Met, and exceeded: the probe went all the way to a working
build under Wine.*

- [x] Baseline measured — 5 compile-blocking sites in 4 packages, 4 layers
- [x] `pkg/xattr` verified: ships `xattr_unsupported.go`, no build tag needed
- [x] These design documents
- [x] `probe/m1-skeleton.patch` — verified to apply cleanly and produce a working `please.exe`
- [x] ~~Non-blocking CI job~~ — overtaken by events. M4 added a *blocking* `build-windows`
      job that builds the whole release, which is strictly stronger
- [x] ~~`go1.27.0.windows-amd64` hash in `third_party/go/BUILD`~~ — **not needed.** Go
      cross-compiles from the host toolchain; there is no Windows distribution to fetch
- [x] `tools/images/windows_builder/Dockerfile`, added to `tools/images/build.sh` — done in M4

### Findings that changed the plan

1. **Errors are layered, not parallel.** `src/process` is a dependency of nearly everything,
   so a single `go build ./...` reports 3 errors and leaves 30 of 51 packages unchecked.
   Build packages independently and iterate.
2. **`syscall.Exec` is not a compile blocker** — Windows ships a stub returning `EWINDOWS`.
   Same for `Chdir` and the signal constants. Silent runtime failures instead.
3. **New: go-flags breaks Please's label syntax on Windows** (D5). `-tags forceposix` fixes
   it. Not in the original plan at all.
4. **`src/output/shell_output.go` is an abstraction leak** the source survey missed.
5. **`path.Dir` in `src/cli/logging.go` is a hard startup blocker**, not a cosmetic bug.
6. **busybox-w64 has a `bash` applet** but **rejects `--noprofile`/`--norc`** — contradicting
   the Linux busybox result. `ShellArgs` is mandatory.

## M1 — OS abstraction layer

**Exit:** `plz build --arch windows_amd64 //src:please` produces `please.exe` via the real
BUILD-file path (not raw `go build`), and `//src/...` unit tests compile. ✅ **Met.**
Full suite: 837 tests, 835 passed, 2 skipped. The cross-built binary parses labels and runs
cold-cache builds under Wine.

Design: `01-os-abstraction.md`. `probe/m1-skeleton.patch` is a starting shape — but its
`lock_windows.go` and `kill_windows.go` are deliberately wrong and must be replaced, not
adopted.

**The compile fixes are ~2 days. The rest of the milestone is the runtime work the compiler
gives no signal for.** Do the cheap fixes first to unblock Wine testing, then the real ones.

- [x] `src/cli/logging.go` `path.Dir` → `filepath.Dir` (hard startup blocker)
- [x] `.plzconfig_windows_amd64` with `[Plugin "go"] BuildTags = forceposix` (D5) —
      `go_binary` has no `tags` param, so this is config, not a BUILD edit
- [x] `src/output/shell_output.go` — leak removed; `process.ShareParentProcessGroup`
- [x] `src/core/lock.go` → `lock_other.go` / `lock_windows.go` (`LockFileEx`), real
      implementation; all 12 lock tests pass under Wine
- [x] `process.ExecReplace` + 5 call sites (the 6th, `sandbox_linux.go`, is Linux-only).
      Verified under Wine: stdout passthrough, exit codes 0 and 3. Also releases the repo
      lock before handing over — on Unix the exec did that implicitly via `O_CLOEXEC`
- [x] `src/process/exec_windows.go` — `CREATE_NEW_PROCESS_GROUP`; job objects in
      `kill_windows.go`
- [x] `src/process/kill_windows.go` / `kill_other.go` — Ctrl-Break then `TerminateJobObject`
- [x] Narrow `exec_other.go` from `!linux` to `!linux && !windows`
- [x] `src/clean/clean.go` — `ForkExec` → detached `exec.Command` (`DETACHED_PROCESS`)
- [x] `src/cli/process.go` — signal set and exit-code convention now per-platform
- [x] `Build.Xattrs` defaults false on Windows; `pkg/xattr` needed no build tag
- [x] `.exe`/`PATHEXT` via `fs.ExecutableNames`, wired into `core.LookPath`.
      Note `isExecutable`'s `0111` check is only reachable on the FreeBSD path, so it needed
      nothing — the design doc over-stated this
- [x] `src/run/run_step.go` — `ExitError.ExitCode()`; `syscall.Chdir` → `os.Chdir`

**Landed early from M3** (M1 is untestable under Wine without it): platform-specific shell
init args, since busybox rejects `--noprofile`/`--norc`. Note this is a property of the shell
being invoked, not the host — remote execution keeps the full flag set via a new
`process.RemoteBashCommand`. The `[build] Shell`/`ShellArgs` *config* is still M3.

## M2 — Paths, environment and the `.exe` model

**Exit:** `//src/core/...` and `//src/fs/...` unit tests pass under Wine, including
`lock_test.go` and config loading.

Design: `01-os-abstraction.md` (the `.exe` model) and `02-shell-and-build-actions.md` (the
path-format rule).

- [x] Promote `splitPathList` → `fs.SplitPathList` (done in M1, needed by `LookPath`)
- [x] Remaining raw `":"` splits in `src/core/config.go` (6) and `src/remote/action.go`.
      The remote one splits locally but still joins with `":"` for the POSIX worker
- [x] `src/fs/home.go` — `os.UserHomeDir()`; `~` regex built from the platform separators
- [x] `MachineConfigFileName` (ProgramData) and `DefaultPath` (empty on Windows — there is
      no equivalent of `/usr/bin` holding build tools)
- [x] `USERPROFILE`, `TEMP`, `TMP` — Windows only, so Unix hashes are untouched
- [x] `src/core/build_target.go` — platform-conditional `SandboxDir`
- [x] `src/fs/copy.go` — symlink privilege fallback (copies the target, warns once)
- [x] `src/fs/fs.go` — `RemoveAll` clears the read-only attribute on files too
- [x] **`glob()` returned nothing at all on Windows** — not on the original list, and fatal.
      `patternToMatcher` built the pattern with `filepath.Join` while the walk goes through
      `io/fs`, whose paths are always slash-separated. Fixed by using `path` throughout
- [x] The raw `"/"` handling in `src/fs/sort.go` and elsewhere in `glob.go` turns out to be
      **correct** for the same reason — `io/fs` paths are always `/`. The design doc was
      wrong to flag them
- [x] Forward-slash normalisation in `BuildEnvironment` + a test asserting no `\`.
      Confirmed by experiment rather than assumption: `echo` and `printf '%s'` round-trip a
      backslash path unharmed, but `sed -e "s#x#$TMP_DIR#"` turned `\tmp` into a literal tab
      and ate the rest — and the cc rules build their link line with `sed`

## M3 — Build actions and the bundled shell ✅

**Exit:** a `genrule` with `cmd = "cat $SRCS | sort > $OUT"` builds under `plz.exe` on Wine.
**Met**, and this time through config and packaging rather than by hand: nothing was placed on
the PATH and no file was renamed to `bash.exe`. The repo has no `[build] shell` line at all;
`plz.exe` finds the `busybox.exe` sitting next to it and runs the action through it.

Design: `02-shell-and-build-actions.md`.

- [x] `[build] Shell` / `ShellArgs` config. Defaults are per-platform, from
      `process.DefaultShell` / `DefaultShellArgs`: `bash` with `--noprofile --norc` on Unix,
      `busybox` with `bash` (the applet name) on Windows
- [x] `src/process/process.go`, `src/run/run_step.go` — `BashCommand` is now a method on
      `Executor`, which carries the shell. `RemoteBashCommand` is untouched and still hardcodes
      the full flag set, because the remote worker is a real bash whatever we are running on
- [x] `src/cache/cmd_cache.go` — both `sh -c` sites use the configured shell
- [x] `src/output/shell_output.go` — `plz build --shell` had a third hardcoded shell, not on
      the original list. It gets `Executor.InteractiveShellCommand`, which is the same shell
      without `-e`/`-u`
- [x] Vendor `busybox.exe` (`remote_file`, pinned hash, GPL-2.0 noted)
- [x] Add to `//package:installed_files` under `is_platform(os = "windows")`
- [x] Applet and flag audit against busybox-w64 — done in M0, see
      `02-shell-and-build-actions.md`
- [x] Gate the `xz -zc` tarball rule to Linux (busybox `xz` is decompress-only — re-verified
      against busybox-w64 1.38.0, which is also decompress-only despite listing the applet).
      `tarball(xzip = True)` now fails at parse time on Windows, and `package/BUILD` defines
      the two xz tarballs only where they can be built
- [x] `plz hash //...` unchanged on Linux — verified by hashing the same tree with the old and
      new binaries, which agree exactly. The config addition is invisible to the hash because
      `Configuration.Hash` covers only `Build.Lang`, `Build.Nonce`, the rejected licences and
      the build environment

**Landed early from M4** (`//package:installed_files` does not build for Windows without it):
`please_sandbox` is gated off Windows. It is built on Linux namespaces, so there was never
anything to ship there, and MinGW rejects `sandbox.c` outright.

### Findings

1. **Resolving the shell on `$PATH` alone would have made the bundling pointless.** Nothing
   puts Please's install directory on the user's PATH on Windows, so a default of `busybox`
   would never have been found. `resolveShell` (`src/core/state.go`) keeps the old behaviour
   for a shell that is on the PATH and falls back to the *build* path — which already has
   `Please.Location` prepended — only when it is not. Verified both ways under Wine: with
   `busybox.exe` beside `plz.exe` the build works with no configuration; with it moved away
   the build fails with `exec: "busybox": executable file not found in %PATH%`.
2. **The `busybox bash` applet form behaves exactly like the `bash.exe` copy M0 tested.**
   Re-verified under Wine: `-e` stops at the first failure (exit 1), `-u` rejects an unset
   variable (exit 2), `-o pipefail` propagates a failure from the left of a pipe (exit 1).
   So `ShellArgs = bash` costs nothing over renaming the binary, and avoids installing a file
   called `bash.exe` that would shadow a user's real bash.
3. **A repeatable config key still cannot be cleared by assigning it empty** — the M5 trap,
   met again. `ShellArgs = ` yields `[""]`, so an empty argument would have been passed
   through to the shell. `ShellArgs` drops empty entries, which is what lets a user turn the
   shell's arguments off entirely.
4. **Bundling busybox needed a licence-policy change.** `.plzconfig` rejected `GPL-2.0`, so
   `//third_party/binary:busybox` failed to build before it was ever run. Accepted with a
   comment: Please execs busybox rather than linking it, so they are separately distributed
   works, but the release has to carry the licence and a pointer to the source.
5. **`plz run` had the same bare-filename bug as `toolPath`** (M6 finding 2), one layer up:
   `run_step.go` decided "does this need looking up on the PATH?" by searching for `/` only,
   so an entry-point path built with `filepath.Join` would have been sent to `LookPath`. Now
   checks `filepath.Separator` too.

## M4 — Release pipeline 🟡

**Exit:** `plz build --arch windows_amd64 //package:release_files` on Linux CI produces a
signed `windows_amd64/` folder. The command **passes locally**; what is left is the arcat
release it depends on for plugins, and running it on CI for real.

Design: `04-release-and-ci.md`.

- [ ] **arcat for `windows_amd64`** — publish the release, then add its hash to
      `src/parse/internal_package.go`. **Upgraded in severity:** it blocks `plugin_repo`, and
      every language plugin is delivered that way, so no plugin can load on Windows without
      it. `cc_library` needs it again for `.a` archives. Simple genrules and parsing work
      without it, which is why this was first recorded as minor.
      Good news: arcat is pure Go with no syscall/cgo, cross-compiles to PE32+, and both
      `arcat x` and `arcat ar -r` verified working under Wine. Its `go.mod` says `go 1.17`
      while the code uses generics, so it fails to build on *any* platform with a modern
      toolchain — a one-line upstream fix, unrelated to Windows.
      **This is the only thing between here and the exit criterion**, and it needs someone with
      push access to the arcat repo. Confirmed still true: the v1.3.1 release has assets for
      darwin, freebsd and linux only.

      **What has been done instead is to stop it being a wall.** Generating the internal
      package used to fail outright on any platform with no published arcat, which stopped
      everything rather than only the things that need one. The arcat rule is simply left out
      now, so the rest of `//_please` still works, and `plz` warns once at startup that anything
      needing arcat — including loading a plugin — will fail unless `[build] arcattool` points
      at a build of your own. That is the difference between "Windows cannot parse anything" and
      "supply this one binary yourself".
- [x] `.plzconfig_windows_amd64` — landed early in M1 (needed for `forceposix`)
- [x] `package/BUILD` — gate `please_sandbox` on `is_platform(os = "linux")` — done in M3
- [x] `package/BUILD` — `.zip` release target, built with `arcat zip` on the Linux release
      box. The two xz tarballs are replaced by it on Windows rather than added to
- [x] `plz.cmd` shim instead of the `ln -sf please plz` symlink
- [x] `src/update/update.go` — `linkFile` is now per-platform. Windows hard-links instead of
      symlinking, and renames a file it cannot replace out of the way to `.stale`, which the
      next run sweeps up in `clean()`
- [x] `pleasew.ps1` + `src/assets/BUILD` + root `BUILD`. **Not executed anywhere yet** —
      there is no PowerShell on the Linux host, so it has been reviewed but not run
- [x] `.circleci/config.yml` — `build-windows` job, workflow entry, `release-gs` requires
- [x] `.circleci/release.sh` — `release_folder … windows_amd64/$VERSION`
- [x] `tools/misc/gen_release.py` — `_arch()` windows branch, and `.zip` added to the
      content-type table
- [x] `tools/images/windows_builder/Dockerfile`, added to `tools/images/build.sh` — the M0
      item. The image tag in `config.yml` is dated `20260910` and **the image has to be built
      and pushed before that job can run**

### What the release turned up

1. **The `.exe` suffix had to be asked for per target.** `go_binary` names the output after
   the rule, so `//src:please` produced a file called `please` for Windows, which cmd will
   not run and `LookPath` will not find. `out = "please.exe" if is_platform(...)` fixes it
   where it matters today; the general fix belongs in the go plugin (M8).
2. **The whole packaging path is verified end to end under Wine.** Extract the zip as a user
   would, run `plz.cmd`, and a genrule builds: the shim finds `please.exe`, which finds
   `busybox.exe` beside it with no configuration at all. `query alltargets //...` also works,
   which is the `forceposix` smoke test the risk register asked for.
3. **`plz init` now writes `pleasew.ps1` as well as `pleasew`**, on every platform. A repo is
   often worked on from more than one, and picking by host would give a Linux developer no way
   to set one up for their Windows colleagues.
4. **The release artifact names carry no extension and that is correct.** `please_<VERSION>`
   is a URL key, not a filename; the updater downloads it and writes it as `please.exe`
   locally. Only the archive members need the suffix.

## Working on the plugins locally

All four plugins are separate repos we have no push access to, so they are cloned into
`~/code/<plugin>-rules` on a `windows` branch each, branched at the tag `plugins/BUILD` pins.

`plugins/BUILD` takes each checkout path from a `[buildconfig]` key, so switching is a
gitignored `.plzconfig.local` and nothing else:

```ini
[buildconfig]
go-rules-path = /home/peter/code/go-rules
```

Delete that file and you are back on the pinned downloads. Verified both directions.

Three things learned wiring this up:

- **`local_repository` is not usable** for this. It omits `plugin = True`, so the subrepo
  registers as `plugins/go` rather than `go` and `///go//...` never resolves. Call the
  `subrepo()` builtin directly.
- **A subrepo `path` outside the repo root is fine.** It is used verbatim with no containment
  check, so no symlink is needed.
- **The local definition has to replace the download, not sit beside it.** The first attempt put
  the `subrepo()` calls in a separate untracked package and pointed `[Plugin "go"] Target` at it.
  That builds fine until anything parses `plugins/BUILD` as well — `plz test //...` does — and
  then dies with *"Found multiple definitions for subrepo 'go'"*. Hence the conditional in the
  one file, which costs a few tracked lines but cannot conflict with itself.

Expect every hash to change when you flip, since the full source path goes into the digest.

## M5 — C++ on Windows (workstream B)

**Exit:** `plz build --arch windows_amd64 //test/...` in cc-rules produces PE32+ `.exe` and
`.dll`.

Design: `03-cc-toolchain.md`. Repo: `please-build/cc-rules`.

- [x] **D1 confirmed.** Both a WinLibs 16.2.0 and an Ubuntu 13 MinGW match the existing GCC
      and GNU ld matchers; the Clang matcher correctly does not. No new matchers needed
- [x] `build_defs/arch.build_defs` — add `windows_amd64` (gates the plugin's own release,
      not its use)
- [x] `cc_binary` / `cc_test` → `.exe`; `cc_shared_object` → `.dll`. **Must be a function,
      not a module-level constant** — subincluded `CONFIG.OS` reflects the host at module level
- [x] A `cc_library` + `cc_binary` + `cc_shared_object` triple builds and `prog.exe` runs
      under Wine, linking the static lib correctly
- [x] Drop `-fPIC` and `-Wl,--build-id=none` for Windows — both were passed and neither
      broke the link, so this is noise reduction rather than a blocker
- [x] `DefaultLdFlags` → `-lpthread`. Only `-ldl` was wrong. Note a repeatable config key
      **cannot be cleared by assigning empty** — that yields `[""]`, which becomes a bare
      `-Wl,` and the linker rejects it.

      **It was set that way anyway**, in *this* repo's `.plzconfig_windows_amd64`, and stayed
      broken because nothing here built a C++ target for Windows until one was added. The
      platform default now comes from the plugin's build defs, where `CONFIG.OS` is the target;
      the plugin's own `.plzconfig_windows_amd64` never applied to anyone using it as a plugin
- [x] **`cc_shared_object` can be linked against.** Windows resolves a DLL's symbols through an
      import library rather than through the DLL, so `-l<name>` had nothing to find. The link
      now writes one with `--out-implib`, named after the output — `lib<name>.dll.a` for the
      default `lib<name>.dll`, which is what `-l<name>` looks for.

      The note in the rule said a second output was impossible, because the shared link command
      names its output `$OUT` and that is unset on a multi-output rule. True of `outs`, not of
      `optional_outs`, which don't count towards it. `//test/windows:dll_test` builds the pair,
      links one against the other and runs it under Wine; taking the DLL away makes it exit 53,
      so the linkage is genuinely dynamic. Windows has no rpath, so the DLL has to sit beside
      the binary — which is what the test rule's data does
- [ ] `please_cc` `execvp_windows.go` (needed for native Windows, not for Axis 2)
- [x] Parse-time error when `pkg_config_libs` is used on Windows, naming the rule that asked
- [ ] MinGW cross-compile job in `plugin_test_cc.yaml`
- [ ] **`please_cc` needs a `windows_amd64` release.** `tools/BUILD` fetches it as a prebuilt
      binary with a pinned hash per platform. Not a blocker under Axis 2, where tools build for
      the Linux host, but required for a native Windows plz
- [ ] **`UnitTest++` does not compile for Windows** as packaged — needs its `Win32/` sources.
      Blocks `cc_test`, not `cc_library`/`cc_binary`
- [ ] Upstream PR; bump `plugins/BUILD` revision

## M6 — Linux-hosted verification harness

**Exit:** one CI job builds and runs a C++ `cc_test` for Windows, from Linux, end to end.

Design: `05-testing-strategy.md`.

- [x] Wine test macro for cross-compiled Go test binaries — `test/build_defs/wine.build_defs`,
      `wine_go_test` and `wine_plz_test`
- [x] Wine CI job — `test-windows-wine`, blocking, and a third pass in `test.sh` where Wine is
      installed. Every `//src/...` package whose tests run there at all: 19 targets, 717 tests,
      787 passing and 7 skipped — **every `//src/...` package**. `//src/build`, `//src/exec`
      and `//src/process` are the valuable ones: they run real build actions and real
      subprocesses, so they cover the layers that are almost entirely separate code on Windows
- [x] The genrule shell smoke test — plus a `query alltargets //...` test, which is the
      `forceposix` guard the risk register asked for
- [x] **The headline end-to-end passes.** `wine plz.exe` extracts the cc plugin with
      `arcat.exe`, runs build actions through busybox, identifies the toolchain with
      `please_cc.exe`, compiles and links with MinGW `g++.exe`, and the resulting `hello.exe`
      runs and prints correctly. Every component in that chain is a Windows binary.

      Two environmental caveats, neither a Please defect: Wine here has no working DNS, so
      the plugin zip has to be supplied locally rather than downloaded; and `arcat` is pointed
      at a cross-built binary via `[build] arcattool`, since there is no published
      `windows_amd64` release yet. Both stand in for release infrastructure, not code.

      Getting there surfaced two real bugs — see the M6 findings below.
- [x] Make the Wine job blocking

### What running the unit tests under Wine surfaced

Doing this properly for the first time found **four correctness bugs**, all of the same shape
and none visible on Linux: code handling repo-relative or label-derived paths through
`filepath`, whose separator on Windows is a backslash. M2 recorded the inverse of this lesson
(`io/fs` paths are always `/`, so use `path`) and fixed the producers; these are the consumers
it missed.

1. **`glob()` crossed package boundaries.** `isBuildFile` called `filepath.Base` on a path from
   `io/fs`, so on Windows it compared the whole path against `BUILD` and never matched. No
   subpackage was ever detected, and a glob in one package would take files belonging to
   another. This is the same function M2 fixed the *pattern* side of.
2. **The initial package was wrong from any subdirectory.** `getRepoRoot` returned it with
   backslashes, which are illegal in a package name, so validation failed and Please walked up
   until something parsed — usually the repo root. `plz build ...` from `src/core` would have
   built the wrong thing silently.
3. **Relative labels didn't parse at all.** `path/to:thingy` became `//current_package\path\to`.
4. **`$(location)`, `$(exe)`, `$(worker)` and tool paths expanded with backslashes** into shell
   commands, where a backslash is an escape character. M2 normalised the *environment*, which is
   a different path.

**This overturned an M2 decision.** `02-shell-and-build-actions.md` argued for normalising only
at the environment boundary because it was the smaller change. It isn't: the replacements above
are not environment values, and the existing tests already assumed forward slashes throughout.
`plz-out` paths are now built with `path`, so they are slash-separated on every platform. Win32
accepts either, and it is a no-op on Unix.

Extending the job past `core` and `fs` found **four more of the same kind**:

5. **`output_dirs` produced doubled paths.** `copyOutDir` strips the temp directory off a path
   to get an output name, comparing a `filepath.Join` result against a slash-separated
   `TmpDir()`. Neither prefix matched, the whole path survived as the output name, and
   `moveOutputs` then joined the temp directory onto a path that already contained it.
6. **JS coverage file names were never sanitised.** `sanitiseFileNameDir` compared paths from a
   coverage file against `filepath.Dir` of a plz-out directory, so coverage was reported
   against absolute build paths instead of source files.
7. **Coverage-by-directory keys came out backslashed**, so they neither read correctly nor
   matched anything configured.
8. **`file://` URLs could not name a Windows path.** RFC 8089 puts a slash before the drive
   letter, so `file:///C:/foo` arrives as `/C:/foo`, which `filepath.IsAbs` rejects. No
   `remote_file` with a local URL could work.

And widening it again, past `build` to the parser, the query layer and the rest, found three
more — including the worst one so far:

9. **`join_path` in the BUILD language returned backslashes on Windows.** The BUILD language is
   the same language on every platform: its paths become labels, sources and command strings,
   and every hash that reaches one would have differed. `subinclude`, `subrepo` names and
   Bazel-compatibility `load()` had the same problem.
10. **Completion returned backslashed package names**, which are not valid build labels, so
    every completion below the top level was unusable.
11. **`plz query changes` matched no package** for a changed file, because it walked up the
    directory tree with `filepath.Dir` and looked the result up as a package name.

And the last packages found a gap in M3's own work:

12. **The command cache never got the resolved shell.** `resolveShell` lived in
    `executorFromConfig`, so only build actions and tests benefited; `[cache] storecommand` and
    `retrievecommand` were handed the bare name and could not find the bundled busybox. It is
    now `Configuration.Shell()`, resolved once and shared, with a final fallback to the
    directory of the running binary — which is where a bundled shell sits, and unlike the build
    path does not depend on `Please.Location` having been resolved yet.
13. **`ShellArgs` was empty for any hand-built configuration.** The defaults for a repeatable
    key can only be applied after parsing, or gcfg appends to them rather than replacing, so
    anything using `DefaultConfiguration()` directly got none. On Unix that quietly dropped
    `--noprofile --norc`; on Windows it dropped the applet name, so the shell did not run at
    all. `Configuration.ShellArgs()` supplies the platform default when nothing is set.

### The job objects from M1, finally under test

`//src/process` used to be excluded because its tests ran `true`, `false` and `sleep` as bare
argv — programs on the PATH on Unix, applets inside the shell on Windows, so only one spelling
works anywhere. Building the argv through the configured shell instead makes them portable, and
that puts the whole process layer under test on Windows for the first time.

`TestKillsProcessTree` is new, and covers what process groups on Unix and job objects on Windows
both exist for: when a command times out, what it started has to die with it. Nothing tested
that on **any** platform before. It fails on Linux if the signal goes to the process rather than
the group.

What the Windows side of it actually proves took some establishing, and the answer is not the
obvious one:

- Disabling `TerminateJobObject` — the test still passes.
- Disabling the Ctrl-Break path as well — the test still passes.
- Disabling `trackProcessTree`, so there is no job object at all — **the run hangs
  indefinitely.**

So it is `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` that does the work, when the handle is closed on
the way out of `ExecWithTimeout`, rather than either explicit kill. And the failure mode without
it is not a surviving grandchild but a hang: the orphans keep Please's pipes open, so it never
finishes rather than failing. That is worth knowing, because a hang is the harder thing to
diagnose in the field.

Note the grandchild has to be a separate process to test any of this. busybox implements a
subshell as a thread on Windows, so `( ... ) &` would die with its parent and prove nothing.

A sweep for the same mistake elsewhere, rather than waiting for a test to find it, turned up
four more places where a path Please *prints* came out backslashed: `plz query outputs`, `plz
query graph`, the paths `plz build` reports, and the entries `plz generate` writes into a
`.gitignore` — the last of which would simply not have matched, since git speaks forward slashes
on every platform. None of these has a test that would notice, so they are worth naming.

A further kind of finding is recorded rather than fixed: **Go's `exec` on Windows will not run a file
whose name has no extension in `PATHEXT`, even given its full path.** The `wine_go_test` macro
copies each test binary to a `.exe` before running it. The same trap is why `//src:please`
needs `out = "please.exe"` (M4), and it will bite `plz run` on any `go_binary` until the go
plugin names Windows outputs properly (M8).

Two things about the harness itself, both found by tests failing for reasons that were nothing
to do with Please:

- **Wine's hosts file has the `localhost` line commented out.** Anything resolving it hangs
  until it gives up — three `remote_file` tests each burned 15 seconds. The macro appends the
  line to the prefix.
- **`~` resolves inside the shared prefix**, so a test writing to its home directory leaks into
  the next run. One of these left a read-only file behind, which on Windows the next run cannot
  replace. Tests that write to `~` now set `USERPROFILE` themselves, which is what
  `os.UserHomeDir` reads there.

Three tests are honestly unrunnable rather than fixed:

- **`TestSymlink` skips on Windows.** Creating a symlink needs Developer Mode, and under Wine
  `os.Symlink` reports success and produces a link that cannot even be `Lstat`ed. The testing
  strategy predicted Wine would grant the privilege unconditionally and so never exercise the
  copy fallback; the reality is worse, and belongs on the M9 agenda.
- **`TestSymlinkedOutputs` and `TestCreatePlzOutGo` skip on Windows**, for the same symlink
  reason, and the permission assertions in `TestOutputDirDoubleStar` skip because Windows has
  no mode to preserve — Go synthesises one from the read-only attribute.

### What the end-to-end test surfaced

1. **No package below the top level parsed on Windows.** `buildFileName` joined with
   `filepath.Join` and then called `iofs.Stat`, so it looked for a file whose *name* contained
   a backslash. Only the root package worked, because `filepath.Join("", "BUILD")` has no
   separator to get wrong — which is exactly why every earlier test missed it. This alone
   would have made Windows unusable for any real repo.
2. **`toolPath` prepended `./` to absolute paths**, producing
   `./Z:/tmp/.../please_cc.exe`, because it decided "is this a bare filename?" by looking for
   `/` only.
3. **`.plzconfig` rejects unquoted backslashes** — `unquoted '\' must be followed by new line
   or double quote`. Windows paths in config files must use forward slashes or be quoted.
   ~~Worth a note in the user docs.~~ **Fixed instead**: the error now names the file it came
   from — it did not before — and says to use forward slashes or quote the value. The parser's
   own message gives no hint that a path is even involved.
4. **`DefaultPath` being empty on Windows is load-bearing**, not cosmetic: `ar.exe not found
   in path` until `[build] path` is configured. That is the intended design, but it means a
   Windows user must configure tool locations before anything builds. **The message now says
   so** when the only directory searched was Please's own, which is exactly the
   nothing-configured state. Note the check counts non-empty entries: clearing a repeatable key
   by assigning it empty yields `[""]`, not an empty list — the same trap as `ShellArgs`, met
   for the third time.

## M7 — Sandboxing parity 🟡

- [x] Default `Sandbox.Build`/`Sandbox.Test` false on Windows, with a clear log line. The
      defaults were already false — the zero value — so the work was the log line, and it
      mattered more than it looks. Setting either on Windows previously produced
      `Can't find sandbox tool please_sandbox on the path`, which invites you to install
      something that does not exist. It now says sandboxing is not implemented on this platform
      and that actions will run without isolation, and does not construct a sandboxing executor
- [ ] `sandbox_windows.go` — Job Objects (reuse M1), restricted token, scrubbed environment
- [x] Document the filesystem-isolation gap. There is no mount-namespace analogue on Windows.
      A job object can bound processes and a restricted token can drop privileges, but neither
      hides a directory, and the only thing that does is a Windows Container — a dependency far
      too large to take on for a build tool. So a Windows sandbox could isolate *processes* but
      not the *filesystem*, which is the half that matters most for build hermeticity. That is
      why refusing to act on the setting, rather than half-implementing it, is the right shape
      until someone has a use for the process half on its own

Note `resolveOut` already guards its sandbox branch on `runtime.GOOS == "linux"`, so `$OUT`
does not change shape on a platform without a sandbox. `target.Sandbox` is still folded into
the target hash, so a Windows user who turns sandboxing on gets different hashes for no
benefit — which is why refusing to act on the setting, rather than quietly ignoring it, is
the right shape.

## M8 — Remote execution and plugin parity 🟡

- [x] ~~`src/remote/action.go` `translateOS` — add `windows`~~ — **nothing to do.**
      `reallyTranslateOS` special-cases only `darwin` → `macos` and passes everything else
      through, so `windows` already comes out as `windows`. Recorded rather than changed
- [ ] go plugin — `windows_amd64` arch, `.exe` naming. **Now blocking more than it looks:**
      Go's `exec` on Windows will not run a file with no `PATHEXT` extension even given its
      full path, so `plz run` on any `go_binary` fails until this lands. `//src:please` and
      `//tools/build_langserver` work around it per-target (M4).

      **Measured, and the constraint is narrower than it appears.** Windows itself runs such a
      file happily — `os.StartProcess` on the bare path works, and prints its output. It is
      only Go's `os/exec` that refuses, in `lookExtensions`, and it refuses even when `Cmd.Path`
      is set directly, so there is no way to keep `os/exec` and bypass it. Reimplementing
      process handling to avoid that is far worse than naming the output correctly, so the fix
      stays with the plugin.

      What is fixed here is the message. `executable file not found in %PATH%` for a file that
      is plainly there is baffling; `fs.ExplainUnrunnable` adds that the name has no extension
      Windows will run, and what it would need to be called
- [x] **go plugin — `.exe` naming done** in the local clone. `go_binary`, `go_test` and
      `go_benchmark` append the suffix from a per-call function. Proof it works: the
      `out = "please.exe" if is_platform(...)` workarounds in `src/BUILD.plz` and
      `//tools/build_langserver` can be deleted and `please.exe` still comes out with the right
      name. **They are deliberately still in the tree**, because this repo pins the unfixed
      upstream plugin; drop them in the same change that bumps `plugins/BUILD`
- [ ] go plugin — `windows_amd64` arch for its own release. `tools/please_go:bootstrap` runs
      `go build ... && mv please_go $OUT`, which fails where `go build` writes `please_go.exe`,
      and hardcodes `TMPDIR=/tmp`. Native-Windows only
- [x] **shell plugin — `sh_test` and `sh_cmd` done.** Windows has no shebang mechanism, so a
      `.sh` is not runnable by name however it is written. `sh_test` hands the script to a shell
      explicitly and `sh_cmd` takes its interpreter from a new `shell_tool` plugin config rather
      than hardcoding `/bin/sh`. Verified under Wine through the bundled busybox

      **Its Windows default was in the wrong place, and did nothing.** It was set in the
      plugin's own `.plzconfig_windows_amd64`, and *a plugin's architecture config is never read
      when it is used as a plugin*: `readSubrepoConfig` reads only `.plzconfig` from the
      subrepo, and the `.plzconfig_<arch>` that `state.ForArch` merges belongs to the repo doing
      the building. So it worked in the plugin's own tests, where it *is* that repo, and a
      Windows user of the plugin silently got `/bin/sh` — a path Windows does not have.

      Reading the subrepo's arch file would not be enough on its own either: a plugin's own
      `[Plugin "x"]` values are not consulted for its config, only `[PluginConfig "x"]
      DefaultValue`, and merging that appends rather than replaces, so a scalar default cannot
      be overridden by a second file. The platform default now lives in the build defs, where
      the rest of the platform handling already is. The python plugin does the same for its
      run-time interpreters, for the same reason
- [x] **shell plugin — `sh_binary` done** in the local clone. It wrote a shebang, appended the
      script, then appended a zip, and relied on the shebang. The payload was never the
      problem, since busybox has `unzip`; only the launching was. It **cannot emit a `.cmd`
      alongside** because `plz run` requires a single output, so on Windows the single output
      *is* the `.cmd`: a four-line batch preamble with the same zip appended after it. cmd.exe
      reads a batch file a line at a time and stops at `exit /b`, so it never reaches the
      archive.

      The script is left inside the zip rather than inlined the way the Unix version inlines
      it. There is no syntax a batch file and a shell script both ignore — a shebang line
      works precisely because it is a comment to the shell — so the preamble unpacks the
      payload and hands the script to `busybox sh`. It sources it rather than running it, so
      that `$0` is the file the user ran, as it is on Unix where the two are one file. Scripts
      find their unpacked dependencies relative to `$0`, and would not otherwise.

      **Two things bit on the way, neither of them about batch files.** Build outputs are
      read-only and the zip preserved that, so the second run of an `sh_binary` could not
      replace what the first one unpacked — Windows forbids it outright — and went on to run
      the stale payload, having complained only on stderr. The rule now zips a *copy* of the
      build directory with the modes relaxed, because the originals are hardlinked to
      `plz-out` and a `chmod` there would quietly make another target's outputs writable.

      And `wine foo.cmd` is not the same thing as running it: what Wine cannot load as a PE it
      hands to the host, so the unfixed `.cmd` — still carrying a Unix shebang — ran under
      `/bin/sh` and printed exactly what the test wanted. `//test/windows:sh_binary_test` goes
      through `cmd.exe` explicitly, runs it twice in the one directory, and fails without
      either fix
- [x] **python plugin — `python_binary` and `python_test` done** in the local clone. A pex is a
      static ELF preamble with a zip appended, so a Windows cross-build produced an ELF-prefixed
      file that was dead on arrival. Four things were needed, and only one of them was the one
      we expected:

      1. **A Windows preamble**, in Go, cross-compiled and embedded in `please_pex` beside the
         native C one; `--os`, defaulting to the build environment's `OS`, picks between them.
         It reads the same configuration from the same place in the archive. It is a separate
         program rather than a port because Windows has no `exec`: it runs the interpreter as a
         child and passes the exit status back, which `_execv` cannot do
      2. **`plz.py` built a regex out of `os.sep`** to match distribution metadata inside the
         zip. Zip member names are always `/`-separated, so on Windows that was a backslash,
         which the regex compiler read as an escape — every pex died on startup, before any of
         its own code ran. The `filepath`-for-`path` mistake again, in Python
      3. **`.pex.exe` naming**, for the same reason `go_binary` needs `.exe`
      4. **A run-time interpreter default of `python` then `py`.** The fallback elsewhere is the
         interpreter that compiled the sources, which cross-compiling makes the host's, and
         `python3` is a spelling a normal Windows install does not have

      **The two-stage plan in the original note was unnecessary.** It assumed the Python inside
      would need work too. It does not: Python skips leading non-zip data, so an ELF-prefixed
      pex already imported and ran correctly under Wine once the `os.sep` bug was fixed. Only
      the launcher was broken, so only the launcher was replaced

      Measured under Wine by `//test/windows:pex_test`, against the embeddable Python from
      python.org: `os.name`, importing the test module out of the zip, reading a data file
      beside it, and importing third-party code through the meta path hook. Exit-code
      propagation is separately guarded by `//test/windows:pex_exit_code_test`, because a
      preamble that always returned 0 would make every failing `python_binary` look fine
- [ ] python plugin — a `please_pex` release carrying the Windows preamble. Until there is one
      this repo builds the tool from the plugin's source, through `PexTool` in
      `.plzconfig_windows_amd64`. Same blocker as `arcat`: no push access, not a porting problem
- [ ] python plugin — `.pyd` extension modules. `SoImport` writes one to a `NamedTemporaryFile`
      and loads it while the handle is still open, which Windows does not allow. Only affects
      pexes containing native wheels; none of the tests here do
- [x] **A parse deadlock in Please itself**, found building the python plugin's `please_pex`
      from source. Resolving a subrepo declared inside another subrepo makes Please look for
      the package that would declare it in the host repo first, and fall back to the subrepo
      that asked when it isn't there. Not being there is the ordinary case, so that lookup
      swallows the error — and it swallowed the parse claim with it, leaving the next caller
      waiting on a parse nobody was going to do. Twelve targets hung with no output at all.
      `//test/subrepo/nested_subrepo_probe` guards it; two subrepos are needed, because the
      first to look is the one that poisons the lookup

- [x] `src/watch` — **this was a bug, not a documentation task.** `plz watch` compares the
      paths it recorded against the ones fsnotify reports. Ours are slash-separated; fsnotify
      on Windows reports backslashes. Nothing matched, so every event was discarded as
      belonging to a file we weren't watching, and the watch would simply never fire — silently,
      since the mismatch looks exactly like an unrelated file changing. Both sides now go
      through `watchKey`. `//test/windows:watch_test` guards it, and fails without the fix

## M9 — Native Windows CI and GA

- [ ] GitHub Actions `windows-latest` job (the only Windows runner available; CircleCI has
      none in this config)
- [ ] Work through the Wine-invisible failures listed in `05-testing-strategy.md`
- [ ] `get_plz.sh` Windows equivalent
- [ ] `README.md`, `docs/faq.html`
- [ ] `docs/milestones/<version>.html` announcement (fragment HTML — see the existing files)
- [ ] `VERSION` bump + `ChangeLog` entry

## Risk register

| Risk | Impact | Mitigation |
|---|---|---|
| MinGW does not match `please_cc`'s existing regexes | Blocks all of M5 | One-command check, first task in M5 |
| ~~busybox-w64 diverges from Linux busybox~~ | **Materialised, resolved.** `--noprofile`/`--norc` rejected | Audit re-run against busybox-w64 in M0; `ShellArgs` promoted from hedge to requirement |
| ~~go-flags `/` option delimiter breaks label syntax~~ | **Found and resolved in M0** | `-tags forceposix` (D5). Must not regress — it is invisible in Please's own source |
| `go_repo` won't generate Windows-only third-party packages on a Linux host | Any unconditional dep on `x/sys/windows` breaks the normal Linux build | Guard such deps with `is_platform(os = "windows")`; `go_library` filters the `_windows.go` srcs to match |
| Assuming `filepath` is always right on Windows | **Materialised twice.** `glob()` matched nothing (M2), then crossed package boundaries and broke relative labels (M6) | Paths from `io/fs`, build labels, and anything going into a shell command are all `/`-separated: use `path`. The inverse of the `logging.go` bug, where `filepath` was the fix. The Wine unit-test job is the guard |
| BUILD files verified only by `go build` | Real breakage invisible until someone runs `plz` | Always verify through `plz build`, not `go build` — this found 3 bugs in one pass |
| Prebuilt per-platform helper binaries with no Windows release | Blocks plugins (arcat) and native cc builds (please_cc) | Both are pure Go and cross-compile cleanly; the work is publishing releases and recording hashes, not porting |
| A dropped `forceposix` tag silently breaks every label | Total CLI breakage, only visible at runtime | ~~Add a Wine smoke test~~ — done: `//test/windows:label_test` |
| Backslash escaping in shell command strings | Intermittent, hard-to-diagnose build failures | The forward-slash rule in `02-shell-and-build-actions.md`, plus an assertion test |
| `.exe` needs to be a core concept after all | Rework of the M2 decision | Verify `plz run` on a `cc_binary` early in M5, before the rest of M5 depends on it |
| Hash drift invalidates every user's cache | Silent, affects all platforms | `plz hash //...` diff on every M1–M3 PR |
| `ERROR_SHARING_VIOLATION` on real Windows | Invisible until M9 | Listed explicitly in the M9 issue; design `RemoveAll` and the updater defensively now |
| arcat platform gate forgotten | `plz.exe` cannot parse anything, discovered late | Called out as a hard gate in M4; it fails at runtime on Windows, not at build time on Linux |
