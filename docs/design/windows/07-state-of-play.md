# State of Play

Status: **Living document** · Last updated: 2026-09-12

Where the Windows port actually is, and what to pick up next. `06-milestones.md` is the
per-milestone tracker with the reasoning; this is the short version for someone starting cold.

## What works today

`please.exe` cross-builds from Linux, runs under Wine, ships busybox as its build shell, and
builds a C++ binary end to end through an entirely Windows toolchain, including a DLL and a
binary linked against it. Python works too: a `python_test` and a `python_binary` both build
for Windows and run there, and so does an `sh_binary`, as a `.cmd` with its payload appended.
The release is a `.zip` containing `please.exe`, `busybox.exe`, `build_langserver.exe` and a
`plz.cmd` shim; extracting it and running `plz.cmd` builds a genrule with no configuration at
all. Built with `bundled-plugins` set it also carries all four plugins and the helper tools, and
then builds an `sh_binary` with the network taken away — see `08-offline-release.md`.

Test coverage on Linux is unchanged and green. Coverage *of Windows behaviour* is 34 targets and
873 tests under Wine, run by a blocking CI job and by `./test.sh` as a third pass. Eight of those
targets only exist when a local plugin checkout is configured — see below.

**And 809 of those tests now run on a real Windows machine, with no Windows-specific skips
left.** The two that still skip there are skipped on every platform and always were. A blocking GitHub Actions job
cross-builds them on Linux and runs them on `windows-latest`, alongside probes that build a repo
with the release zip, clean and rebuild it five times, and build at a long path. That job is the
only thing anywhere that is not taking Wine's word for it.

| # | Milestone | State |
|---|---|---|
| M0–M3, M6 | baseline, OS layer, paths, shell, Wine harness | done |
| M4 | release pipeline | done; `arcat` is built from source rather than downloaded |
| M4a | offline release zip | done, for internal use — see `08-offline-release.md` |
| M5 | C++ / cc-rules | done, `cc_test` included |
| M7 | sandboxing | decided against, documented |
| M8 | plugins | go, cc, shell, python all done in local clones |
| M9 | native Windows CI and GA | CI done and blocking; GA not started |

## The five repos

| Repo | Branch | Head |
|---|---|---|
| `~/code/please` | `wine` | merged to `master` on the fork |
| `~/code/go-rules` | `windows` | don't double the `.exe` |
| `~/code/cc-rules` | `windows` | emit an import library |
| `~/code/shell-rules` | `windows` | build an `sh_binary` as a `.cmd` |
| `~/code/python-rules` | `windows` | build a `.pex` Windows can run |

The plugin clones are branched at the tag `plugins/BUILD` pins, not at `master`. There is no
push access to any of the *upstream* repos, so nothing is upstreamed, but all five are pushed to
forks at `PeterNeiss/{please,go-rules,cc-rules,shell-rules,python-rules}` and that is where the
branches live.

`.plzconfig.local` (gitignored) selects the local checkouts through `[buildconfig]` keys —
`go-rules-path` and friends, plus `bundled-plugins` to put them in the release. Delete it to go back to the pinned downloads. Both directions are
verified, but they are not equivalent any more. Six Wine tests are only *defined* when the
matching checkout is configured, because no released plugin has the fix each one tests: two pex
tests behind `python-rules-path`, the DLL and `cc_test` tests behind `cc-rules-path`, the
`sh_binary` test behind `shell-rules-path`, and the three offline-release tests behind
`bundled-plugins`. `//test/export:...` fails while `.plzconfig.local` is present at all,
for an unrelated reason — see below.

## Environment

- Wine prefix: `/tmp/claude-1000/-home-peter-code-please/<session>/scratchpad/wineprefix`. It is
  session-scoped, so a new session recreates it with `wineboot --init`; the test macros do this
  themselves under `plz-out/wineprefix`.
- MinGW is installed (`x86_64-w64-mingw32-g++`), which is what cross-builds C++ for Windows.
- **`plz` on the PATH is not this repo's Please**, and the difference is not cosmetic: the
  installed one predates the parse-deadlock fix, so running the Wine tests with it hangs at
  the end of the parse with no error. Build `//src:please` and run `plz-out/bin/src/please`,
  or `./test.sh`, which does that itself. `plz install` also settles it.
- `go` is not on the default PATH. Use `export PATH="$PWD/plz-out/bin/third_party/go/toolchain/bin:$PATH"`
  before `plz lint` or `./test.sh`.
- The `BUILD` files this repo already had are not `plz fmt` clean. Format only the files you
  touch, or you will bury your diff.

## Pick up here

In rough order of value.

1. **Ctrl-Break is delivered but never verified.** `KillProcess` sends one, waits 30ms, then
   terminates the job object. `TestKillsProcessTree` passes natively, but it only asserts a
   grandchild died, which terminating the job achieves either way — so the graceful path could
   be dead code on Windows and no test would notice.

   Harder than it looks, which is why it is still here. The window is 30ms: `KillProcess` sends
   the break, waits that long, then terminates the job regardless. A test that asserts the child
   shut down gracefully is racing that timer on a CI machine, and a flaky test in a blocking job
   is worse than no test. Either call `killProcessTree` directly and wait generously, which
   tests the delivery without the timer, or widen the window and say why.
2. **`sh_test` cannot take an `sh_binary` as its `src` on Windows.** It copies whatever it is
   given to `<name>.sh` and hands that to a shell, and a `.cmd` is not a shell script. The
   plugin's own tests are written that way, so they are the thing to fix it against.
3. **Bump `plugins/BUILD`** once the plugin branches are published somewhere. In the same change,
   delete everything this repo carries because it pins plugins without the fixes: the
   `out = "please.exe" if is_platform(...)` workarounds in `src/BUILD.plz` and
   `//tools/build_langserver`, the `PexTool` and `defaultldflags` lines in
   `.plzconfig_windows_amd64`, the `CONFIG.get(...)` conditions around the pex, DLL and
   `sh_binary` tests in `//test/windows`, and the whole of `08-offline-release.md`'s machinery.
4. **`.pyd` extension modules in a pex.** `SoImport` writes one to a `NamedTemporaryFile` and
   loads it while the handle is still open, which Windows does not allow. Only bites a pex
   containing native wheels.
5. **`plz debug` and `plz cover` on a Windows target** are untested. So is `plz cover`, whose
   coverage paths come back from the Python side with backslashes in them. `plz run` on an
   `sh_binary` is the interesting case: Go's `os/exec` launches a `.cmd` happily under Wine,
   which is the part that was in doubt, and is exactly the kind of answer Wine gives more
   readily than Windows does.

Blocked on push access we do not have: publishing `windows_amd64` releases of `arcat`,
`please_go`, `please_cc`, and a `please_pex` of any platform carrying the Windows preamble.
**None of that blocks cross-building** — tools resolve to the host under `--arch` — it blocks a
*native* Windows `plz` only.

## Things that will bite you again

Each of these has already cost time once.

- **`filepath` is wrong for anything that is a build label, a plz-out path, or destined for a
  shell command.** Use `path`. This produced roughly a dozen bugs across the port, including
  globs crossing package boundaries and `join_path` in the BUILD language returning backslashes,
  which would have changed every hash that reached it. It has a Python dialect too: `os.sep` in
  the pex bootstrap, matching zip member names, which are always `/`-separated.
- **In a plugin's build defs, a suffix must come from a function, not a module-level constant.**
  At module level `CONFIG.OS` is the host, so a constant passes on Linux and misnames everything
  when cross-compiling. Make the function idempotent while you are there: appending `.exe` to a
  name that already ends in `.exe` produced `please.exe.exe`, which nothing could find.
- **A plugin's own `.plzconfig_<arch>` is never read when it is used as a plugin.** Only
  `.plzconfig` is read from the subrepo, and the arch file that gets merged belongs to the repo
  doing the building. A platform default set that way works in the plugin's own tests and
  nowhere else. Put it in the build defs instead.
- **A repeatable config key cannot be cleared by assigning it empty** — that yields a list of one
  empty string. Four separate bugs so far. The most recent sat in this repo's own
  `.plzconfig_windows_amd64` for weeks, because nothing here built a C++ target for Windows
  until a test did.
- **Go's `os/exec` will not run a file with no `PATHEXT` extension**, even given its full path.
  Windows itself is fine with it; `os.StartProcess` proves that. Only the lookup refuses.
- **Never run a cross-built test binary by hand in the source tree.** Under `plz test` they get a
  sandboxed temp directory; run from the repo root they operate on the repo. Doing this once
  deleted the whole of `test/`.
- **`//test/export:...` fails whenever `.plzconfig.local` is present.** The local checkouts are
  registered with `subrepo()` rather than `plugin_repo()`, so there is no target for `plz export`
  to follow and the exported repo has no `plugins/BUILD`. Nothing to do with the port; move the
  file aside before believing an export failure.
- **A build output is read-only, and on Windows that means it cannot be replaced at all.**
  Unpacking an archive of build outputs over a previous unpacking of itself therefore fails,
  and tools tend to report it on stderr and carry on with the stale copy. `sh_binary` hit this;
  anything else that unpacks build outputs beside themselves will too.
- **A skip hides a bug better than a missing test does.** Deleting two has now found two real
  failures that Wine had passed for months. `plz run` handed `cmd.exe` a forward-slashed path,
  which it reads as a switch; and every `link:` label silently became a warning. Both were
  behind `runtime.GOOS == "windows"` skips that looked reasonable when they were written.
- **`filepath.Split` does not terminate a walk on Windows.** Trimming the separator off `C:\`
  leaves `C:`, and splitting that returns it unchanged, so a loop that stops at an empty string
  never stops. Compare each step against the previous one instead. This hung every `plz` run
  outside a repo, at 100% CPU, and nothing predicted it.
- **A handle this process holds is still a handle.** Windows will not rename or delete a
  directory containing a file anything has open, including us. The log file lives under
  `plz-out` by default, which is what `plz clean` deletes.
- **`upload-artifact` drops hidden files** unless `include-hidden-files` is set. A dotfile that
  exists in `plz-out` is silently not in the artifact, and nothing on the Linux side can catch
  it, because the Wine tests never go through one.
- **`plz-out/pkg` is never refreshed once it exists.** The `hlink:` label goes through
  `fs.LinkIfNotExists`, and the destination is named after the version, so rebuilding a release
  at the same version leaves the previous bytes there, silently. `plz-out/gen/<arch>/package/`
  always has the real artifact. Affects every platform; cost an hour here, twice.
- **`chmod` in a build directory writes through to `plz-out`.** Inputs are hardlinked in, so
  relaxing a mode there silently makes another target's outputs writable. Copy first if the
  modes need changing.
- **`wine foo.cmd` does not run it as Windows would.** What Wine cannot load as a PE it hands
  to the host, so a `.cmd` that still has a Unix shebang on it runs under `/bin/sh` and passes
  the test you wrote to catch exactly that. Go through `cmd.exe` explicitly.
- **`plz update` on Windows fetches only the bare binary, not the zip.** Everything else the
  release ships - busybox, and anything `08-offline-release.md` adds beside it - stays at the
  version it was first installed at, silently, getting staler with each update. Nothing has
  ever exercised this.
- **Python under Wine needs its output to be a pipe.** Wine's console emulation hands it handles
  it rejects at startup otherwise, and the error — `can't initialize sys standard streams` — reads
  like a problem with whatever you were testing. It is not.
