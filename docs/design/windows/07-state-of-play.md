# State of Play

Status: **Living document** · Last updated: 2026-09-10

Where the Windows port actually is, and what to pick up next. `06-milestones.md` is the
per-milestone tracker with the reasoning; this is the short version for someone starting cold.

## What works today

`please.exe` cross-builds from Linux, runs under Wine, ships busybox as its build shell, and
builds a C++ binary end to end through an entirely Windows toolchain. The release is a `.zip`
containing `please.exe`, `busybox.exe`, `build_langserver.exe` and a `plz.cmd` shim; extracting
it and running `plz.cmd` builds a genrule with no configuration at all.

Test coverage on Linux is unchanged and green. Coverage *of Windows behaviour* is 25 targets and
802 tests under Wine, run by a blocking CI job and by `./test.sh` as a third pass.

| # | Milestone | State |
|---|---|---|
| M0–M3, M6 | baseline, OS layer, paths, shell, Wine harness | done |
| M4 | release pipeline | done bar a published `arcat` |
| M5 | C++ / cc-rules | rules done; `cc_test` and linking against a DLL remain |
| M7 | sandboxing | decided against, documented |
| M8 | plugins | go, cc, shell done in local clones; python not started |
| M9 | native Windows CI and GA | not started |

## The five repos

| Repo | Branch | Head |
|---|---|---|
| `~/code/please` | `wine` | 45 commits ahead of `master` |
| `~/code/go-rules` | `windows` | `.exe` naming |
| `~/code/cc-rules` | `windows` | build for Windows |
| `~/code/shell-rules` | `windows` | run scripts through a shell |
| `~/code/python-rules` | `windows` | unchanged, at `v2.0.2` |

The plugin clones are branched at the tag `plugins/BUILD` pins, not at `master`. We have no push
access to any of them, so nothing is upstreamed; the branches are the deliverable for now.

`.plzconfig.local` (gitignored) selects the local checkouts through `[buildconfig]` keys —
`go-rules-path` and friends. Delete it to go back to the pinned downloads. Both directions are
verified.

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

1. **python plugin — not started.** The shape is not what it looks like: a pex is a static ELF
   preamble with a zip appended, not a shebang script, so a Windows cross-build produces a file
   that is dead on arrival. Two stages, in `06-milestones.md` under M8: skip the preamble and
   pass the interpreter in `test_cmd`, which gets `python_test` working cheaply; then a small Go
   preamble cross-compiled for Windows for `python_binary`. A C port was rejected — Windows has
   no true `exec`, so `_execv` breaks exit codes and console attachment.
2. **`cc_shared_object` cannot be linked against on Windows.** `-l<name>` needs an import library
   describing the DLL's exports. Declaring one makes the rule multi-output, which breaks the
   command template that names its output `$OUT`. Noted in `cc.build_defs` where it bites.
3. **`cc_test` is blocked upstream of us** — `UnitTest++` as packaged needs its `Win32/` sources
   to compile at all.
4. **`sh_binary`** writes a shebang, appends the script, then appends a zip, and relies on the
   shebang. The payload is fine (busybox has `unzip`); only the launching is broken, and it
   cannot emit a `.cmd` alongside because `plz run` requires a single output.
5. **Bump `plugins/BUILD`** once the plugin branches are published somewhere, and in the same
   change delete the `out = "please.exe" if is_platform(...)` workarounds from `src/BUILD.plz`
   and `//tools/build_langserver`. They are deliberately still there, because this repo pins the
   unfixed upstream go plugin.

Blocked on push access we do not have: publishing `windows_amd64` releases of `arcat`,
`please_go`, `please_cc` and `please_pex`. **None of that blocks cross-building** — tools resolve
to the host under `--arch` — it blocks a *native* Windows `plz` only.

## Things that will bite you again

Each of these has already cost time once.

- **`filepath` is wrong for anything that is a build label, a plz-out path, or destined for a
  shell command.** Use `path`. This produced roughly a dozen bugs across the port, including
  globs crossing package boundaries and `join_path` in the BUILD language returning backslashes,
  which would have changed every hash that reached it.
- **In a plugin's build defs, a suffix must come from a function, not a module-level constant.**
  At module level `CONFIG.OS` is the host, so a constant passes on Linux and misnames everything
  when cross-compiling.
- **A repeatable config key cannot be cleared by assigning it empty** — that yields a list of one
  empty string. Three separate bugs so far.
- **Go's `os/exec` will not run a file with no `PATHEXT` extension**, even given its full path.
  Windows itself is fine with it; `os.StartProcess` proves that. Only the lookup refuses.
- **Never run a cross-built test binary by hand in the source tree.** Under `plz test` they get a
  sandboxed temp directory; run from the repo root they operate on the repo. Doing this once
  deleted the whole of `test/`.
