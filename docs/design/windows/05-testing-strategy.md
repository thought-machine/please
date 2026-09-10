# Testing Strategy

Status: **Draft** · Milestone: M6 (with M9 as the follow-up) · Last updated: 2026-09-10

The programme constraint is that development and CI stay on Linux, with real Windows testing
deferred. This document is how that is made to work rather than merely asserted.

## Three test loops

| Loop | Runs | Tests | Available from |
|---|---|---|---|
| **A — compile gate** | Linux, natively | Does `plz.exe` build for `GOOS=windows`? | M0 |
| **B — C++ cross-build** | Linux, natively | Do the cc rules produce correct PE32+ artifacts? | M5 |
| **C — Wine** | Linux, under Wine | Does `plz.exe` actually *run*? | M1 onwards |

Loops A and B need no emulation at all. Loop C is where the leverage is, and it is why M6
should start as soon as M1 produces a binary — the milestone number is a completion point,
not a start date.

## Loop A — compile gate

```bash
plz build --arch windows_amd64 //src:please
```

Wired into CI as a non-blocking job in M0 (see `04-release-and-ci.md`). Its output is the
burn-down list in `appendix-baseline-errors.md`.

Cheap, fast, and catches the majority of M1's work. It catches nothing about behaviour.

## Loop B — C++ cross-build (Axis 2)

```bash
plz build --arch windows_amd64 //test/cc/...
file plz-out/bin/windows_amd64/test/cc/binary.exe
# PE32+ executable (console) x86-64, for MS Windows
```

`plz` runs as a native Linux binary; `x86_64-w64-mingw32-g++` runs as a native Linux binary;
the *output* is Windows. Nothing is emulated.

**This validates roughly 80% of the C++ work**: flag assembly, output naming, the
`please_cc` tool-identification path, archive combination, transitive label propagation, and
the whole `cc_library` → `cc_binary` graph. All of it before a Windows machine exists.

What it does not validate: that the same flags are produced when `plz` itself is running on
Windows with the bundled busybox shell. That needs Loop C.

Assertions worth making beyond `file(1)`:

- `x86_64-w64-mingw32-objdump -p out.exe` — check the import table names the expected DLLs.
- `x86_64-w64-mingw32-nm` on the archive — check `--whole-archive` actually pulled symbols in
  for an `alwayslink` library.
- Run the produced `.exe` under Wine — which is Loop C applied to the *output* rather than to
  `plz`.

## Loop C — Wine

**Proven, not hypothetical.** Wine 9.0 (the Ubuntu/Pop!_OS package, `apt install wine64`)
already runs `please.exe` end to end: version, `query alltargets //...`, and real builds
including shell pipelines. See `appendix-baseline-errors.md`.

Wine implements precisely the primitives the port introduces: Job Objects, `LockFileEx`,
`CreateProcess`, console control events, `PATHEXT` resolution. That is not a coincidence —
they are the well-trodden Win32 core, which is what Wine covers best.

Setup used:

```bash
export WINEPREFIX=$PWD/.wineprefix WINEDEBUG=-all
wineboot --init
# busybox-w64 as the build shell
cp busybox64.exe wbin/bash.exe
export WINEPATH='Z:\path\to\wbin'
```

`WINEDEBUG=-all` suppresses Wine's chatter; a job-local `WINEPREFIX` keeps runs isolated.

**Clear Please's cache between runs, or you will verify nothing.** `rm -rf plz-out` is not
enough: Please also keeps a directory cache, which under Wine lands in
`$WINEPREFIX/drive_c/users/<user>/AppData/Local/please`. A build that appears to succeed may
be replaying cached artifacts from an earlier, differently-built binary — this happened during
M0 and produced a false pass on a binary whose shell handling was in fact broken. Wipe both:

```bash
rm -rf plz-out "$WINEPREFIX"/drive_c/users/*/AppData/Local/please
```

(Incidentally this confirms `os.UserCacheDir()` resolves correctly on Windows.)
Note Wine maps Unix paths to the `Z:` drive, so `$PWD` becomes `Z:\...` inside the binary —
useful to know when reading error messages.

```bash
wine plz-out/bin/windows_amd64/src/please.exe --version
wine plz-out/bin/windows_amd64/src/please.exe build //test/cc:binary
wine plz-out/bin/windows_amd64/test/cc/binary.exe
```

### What to run under it

1. **Unit tests.** Add a test macro that runs a Go test binary cross-compiled for Windows
   under `wine`. The high-value packages are exactly the ones M1 and M2 touch:
   `src/core` (`lock_test.go`, config loading), `src/fs`, `src/process`.
2. **The genrule smoke test** from `02-shell-and-build-actions.md` — a `cmd` with a pipe and
   a redirect, proving the bundled shell is wired up.
3. **The headline end-to-end test**, combining both axes: `wine plz.exe` building a C++
   project with MinGW. This is the M6 exit criterion and the single most valuable test in the
   programme, because it is the first thing that exercises busybox, the generated cc command
   lines, the Windows process layer and PE output *together*.

### CI

**Implemented.** `test/build_defs/wine.build_defs` has two macros — `wine_go_test` for a
cross-built Go test binary, and `wine_plz_test` for `please.exe` driving a small repo laid out
the way the release is. `//test/windows` uses them; the `test-windows-wine` CircleCI job runs
them and blocks the release.

They are labelled `wine` and excluded from the other test passes, because building them means
cross-compiling the Go standard library for another platform. `test.sh` runs them as a third
pass where Wine is installed.

Two things learned building it:

- **Rename the binary to `.exe` first.** Go's `exec` on Windows will not run a file whose name
  has no extension in `PATHEXT`, even given its full path, and the go plugin names test
  binaries after the rule. Any test whose subject re-execs itself — `TestComplete` in
  `src/core` does — fails obscurely otherwise.
- **A `go_test`'s own `data` doesn't come with it** when another rule depends on the binary, so
  anything the test reads has to be repeated on the `wine_go_test`.

A Wine job on `ubuntu-latest` (`apt-get install wine64`) is cheap. Make it blocking once M1
lands — the whole point is to catch Windows regressions from contributors who are not
thinking about Windows.

Set `WINEDEBUG=-all` to suppress Wine's chatter, and `WINEPREFIX` to a job-local directory so
the prefix is not shared between runs.

## What Wine does not cover

Be honest about this. Wine passing is evidence, not proof. These are the M9 agenda, and they
should be listed in the M9 issue rather than discovered during it.

### Filesystem semantics

- **Case-insensitivity.** Wine on ext4 is case-*sensitive* by default. A BUILD graph with
  `Foo.h` and `foo.h` works under Wine and collides on NTFS. Please's glob and hash code has
  no case-folding anywhere.
- **`ERROR_SHARING_VIOLATION`.** Windows refuses to delete or rename a file that another
  process has open. Wine is more permissive. This is the single most likely source of
  real-Windows-only failures, and it hits exactly where Please works hardest: `plz-out/tmp`
  teardown, `RemoveAll`, and the self-updater overwriting a running binary.
- **`MAX_PATH`.** 260 characters unless long-path support is enabled *and* the binary has the
  manifest opt-in. `plz-out/bin/<subrepo>/<pkg>/<target>` nests deeply; a monorepo will hit
  this. Wine does not enforce it.
- **Symlink privileges.** `os.Symlink` needs Developer Mode or
  `SeCreateSymbolicLinkPrivilege`. This entry predicted Wine would grant it unconditionally, so
  that the M2 copy-fallback path would never be exercised. **Measured, and it is worse than
  that:** Wine's `os.Symlink` returns no error and produces a link that `os.Lstat` then cannot
  find. `TestSymlink` skips on Windows for that reason. So Wine tells us nothing either way
  here, and the fallback still needs testing by injecting a failure, not by hoping.

### Process and console

- **Real console behaviour.** VT sequence support (`golang.org/x/term`), the interactive
  display in `src/output/interactive_display.go`, window resize. Wine's console is not
  conhost.
- **Ctrl-C / Ctrl-Break delivery.** Wine's `GenerateConsoleCtrlEvent` is approximate. The
  graceful-then-forceful kill path in `01-os-abstraction.md` needs native verification.
- **Antivirus.** Real-time scanning locks freshly written executables, causing intermittent
  `ERROR_SHARING_VIOLATION` and slow builds. Invisible under Wine and a genuine user-facing
  problem — worth a documented note in the eventual user docs.

### Toolchain

- **MSVC**, when it arrives, cannot be tested under Wine at all.

## Regression protection for the platforms that already work

Every change in M1–M3 touches shared code paths. Two things must hold for every PR:

```bash
./bootstrap.sh    # full build + unit + e2e on Linux
plz lint          # golangci-lint + plz fmt check
```

**Hash stability is a hard requirement.** Please's cache is content-hash based over rule
definition, config, sources and secrets. Any change to command generation, environment
variables or config defaults shifts target hashes and invalidates every user's cache. Before
merging M3 in particular:

```bash
plz hash //src/...                 # record
git stash -u -- src && plz hash //src/...   # record again at HEAD
git stash pop
```

**Compare within one working directory.** Hashes are *not* comparable between two checkouts
of the same commit — a `git worktree` at HEAD produces different hashes from the main repo
for reasons unrelated to any change, so a worktree-vs-repo diff reports dozens of false
positives. Stash and unstash in place instead.

Expect the dependency cone of whatever you edited to change; that is content hashing working.
What matters is that nothing *outside* that cone moves.

A diff here is not necessarily wrong, but it must be *intended* and called out in the PR
description.

**The e2e tests in `test/` assert on exact output text** and are documented as brittle. Expect
to update `.txt` golden files. Treat any *unexpected* change as a real regression rather than
noise — that is what they are there for.

## Exit criterion for M6

A single CI job, on Linux, that:

1. cross-builds `please.exe`,
2. cross-builds a C++ project for Windows using MinGW,
3. runs `please.exe` under Wine to drive that build,
4. runs the resulting `cc_test` binary under Wine and collects its results.

If that passes, the Windows port is real, and M9 is about hardening rather than discovery.
