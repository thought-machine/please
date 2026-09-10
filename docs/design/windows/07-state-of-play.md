# State of Play

Status: **Living document** · Last updated: 2026-09-10

Where the Windows port actually is, and what to pick up next. `06-milestones.md` is the
per-milestone tracker with the reasoning; this is the short version for someone starting cold.

## What works today

`please.exe` cross-builds from Linux, runs under Wine, ships busybox as its build shell, and
builds a C++ binary end to end through an entirely Windows toolchain, including a DLL and a
binary linked against it. Python works too: a `python_test` and a `python_binary` both build
for Windows and run there. The release is a `.zip` containing `please.exe`, `busybox.exe`,
`build_langserver.exe` and a `plz.cmd` shim; extracting it and running `plz.cmd` builds a
genrule with no configuration at all.

Test coverage on Linux is unchanged and green. Coverage *of Windows behaviour* is 28 targets and
808 tests under Wine, run by a blocking CI job and by `./test.sh` as a third pass. Three of those
targets only exist when a local plugin checkout is configured — see below.

| # | Milestone | State |
|---|---|---|
| M0–M3, M6 | baseline, OS layer, paths, shell, Wine harness | done |
| M4 | release pipeline | done bar a published `arcat` |
| M5 | C++ / cc-rules | done bar `cc_test`, which is blocked upstream |
| M7 | sandboxing | decided against, documented |
| M8 | plugins | go, cc, shell, python done in local clones; `sh_binary` remains |
| M9 | native Windows CI and GA | not started |

## The five repos

| Repo | Branch | Head |
|---|---|---|
| `~/code/please` | `wine` | 52 commits ahead of `master` |
| `~/code/go-rules` | `windows` | don't double the `.exe` |
| `~/code/cc-rules` | `windows` | emit an import library |
| `~/code/shell-rules` | `windows` | default the shell from the build defs |
| `~/code/python-rules` | `windows` | build a `.pex` Windows can run |

The plugin clones are branched at the tag `plugins/BUILD` pins, not at `master`. We have no push
access to any of them, so nothing is upstreamed; the branches are the deliverable for now.

`.plzconfig.local` (gitignored) selects the local checkouts through `[buildconfig]` keys —
`go-rules-path` and friends. Delete it to go back to the pinned downloads. Both directions are
verified, but they are not equivalent any more. Three Wine tests are only *defined* when the
matching checkout is configured, because no released plugin has the fix each one tests: two pex
tests behind `python-rules-path`, and the DLL test behind `cc-rules-path`. `//test/export:...`
fails while `.plzconfig.local` is present at all, for an unrelated reason — see below.

## Environment

- Wine prefix: `/tmp/claude-1000/-home-peter-code-please/<session>/scratchpad/wineprefix`. It is
  session-scoped, so a new session recreates it with `wineboot --init`; the test macros do this
  themselves under `plz-out/wineprefix`.
- MinGW is installed (`x86_64-w64-mingw32-g++`), which is what cross-builds C++ for Windows.
- `go` is not on the default PATH. Use `export PATH="$PWD/plz-out/bin/third_party/go/toolchain/bin:$PATH"`
  before `plz lint` or `./test.sh`.
- The `BUILD` files this repo already had are not `plz fmt` clean. Format only the files you
  touch, or you will bury your diff.

## Pick up here

In rough order of value.

1. **`cc_test` is blocked upstream of us** — `UnitTest++` as packaged needs its `Win32/` sources
   to compile at all. It is the last thing in M5.
2. **`sh_binary`** writes a shebang, appends the script, then appends a zip, and relies on the
   shebang. The payload is fine (busybox has `unzip`); only the launching is broken, and it
   cannot emit a `.cmd` alongside because `plz run` requires a single output.
3. **Bump `plugins/BUILD`** once the plugin branches are published somewhere. In the same change,
   delete everything this repo carries because it pins plugins without the fixes: the
   `out = "please.exe" if is_platform(...)` workarounds in `src/BUILD.plz` and
   `//tools/build_langserver`, the `PexTool` and `defaultldflags` lines in
   `.plzconfig_windows_amd64`, and the `CONFIG.get(...)` conditions around the pex and DLL tests
   in `//test/windows`.
4. **`.pyd` extension modules in a pex.** `SoImport` writes one to a `NamedTemporaryFile` and
   loads it while the handle is still open, which Windows does not allow. Only bites a pex
   containing native wheels.
5. **`plz run` and `plz debug` on a Windows target** are untested. So is `plz cover`, whose
   coverage paths come back from the Python side with backslashes in them.

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
- **Python under Wine needs its output to be a pipe.** Wine's console emulation hands it handles
  it rejects at startup otherwise, and the error — `can't initialize sys standard streams` — reads
  like a problem with whatever you were testing. It is not.
