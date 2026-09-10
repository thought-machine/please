# OS Abstraction Layer

Status: **Draft** · Milestones: M1, M2 · Last updated: 2026-09-10

How platform-specific code is organised, and the design of each abstraction the Windows
port introduces. See `00-overview.md` for the programme charter.

## The convention

Please already has the pattern; it is just barely used. `src/cli/winch_windows.go` plus
`src/cli/winch_other.go` (`//go:build !windows`) is the template. Follow it:

```
foo_windows.go     // no build tag needed — the filename suffix is the constraint
foo_other.go       // //go:build !windows
```

Two rules that are easy to get wrong here:

1. **The filename suffix is itself a build constraint.** `exec_linux.go` has no `//go:build`
   line and does not need one. Adding a redundant one is harmless; omitting the constraint
   on the `_other.go` sibling is not.
2. **`plz` lists `srcs` explicitly in BUILD files** (see `src/process/BUILD`). New files
   must be added there. `plz puku sync` handles `third_party/go`, not first-party srcs.

### Beware the `!linux` files

`src/process/exec_other.go` and `src/sandbox/sandbox_other.go` are constrained `!linux`,
which means **they are selected on Windows**. One of them compiles there and one does not:

- `sandbox_other.go` degrades to a plain `exec.Command(...).Run()` and compiles fine. Not a
  blocker.
- `exec_other.go` sets `SysProcAttr{Setpgid, Foreground}` — fields that do not exist in
  Windows' `SysProcAttr`. Narrow its constraint to `!linux && !windows`.

## Inventory

**Measured, not predicted** — see `appendix-baseline-errors.md` for method and evidence.
The original source survey over-stated this considerably.

### Blocks compilation — the complete set

Four layers, five sites, four packages. Each layer is only visible once the previous one is
fixed, because `src/process` is a dependency of nearly everything.

| Layer | Site | Problem | Design |
|---|---|---|---|
| 1 | `src/process/exec_other.go:17,18` | `SysProcAttr{Setpgid, Foreground}` | New `exec_windows.go`; narrow `!linux` → `!linux && !windows` |
| 1 | `src/process/process.go:206` | `syscall.Kill(-pid, …)` | Job Objects — see below |
| 2 | `src/core/lock.go` ×10 | `syscall.Flock`, `LOCK_SH/EX/UN/NB` | `LockFileEx` — see below |
| 3 | `src/clean/clean.go:96` | `syscall.ForkExec` | Detached `exec.Command` |
| 3 | `src/output/shell_output.go:467` | `cmd.SysProcAttr.Setpgid` | **Abstraction leak** — see below |

After these, every package compiles and `./src` links to a valid PE32+ binary.

### The abstraction leak

`src/output/shell_output.go:467` reaches into the process executor's platform-specific
attributes from *outside* `src/process`:

```go
cmd := state.ProcessExecutor.ExecCommand(...)
// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
cmd.SysProcAttr.Setpgid = false
```

The fix is not a build tag at the call site — it is to expose the *intent* from `src/process`
(`process.ClearProcessGroup(cmd)`, or a parameter on `ExecCommand`) so the platform detail
stays in one package. Audit for other instances while doing M1.

### Does NOT block compilation (corrections)

Three predictions were wrong, all from the same mistake: assuming "Unix-only API" means "does
not compile on Windows". Go's `syscall` package ships Windows stubs.

| Site | Reality |
|---|---|
| `syscall.Exec` ×5 (`src/please.go`, `src/run`, `src/tool`, `src/update`, `tools/please_shim`) | Defined in `syscall/exec_windows.go`, returns `EWINDOWS`. **Compiles; fails at runtime.** |
| `syscall.Chdir` (`src/run/run_step.go`) | Defined on Windows |
| `SIGHUP`/`SIGQUIT`/`SIGABRT` (`src/cli/process.go`) | All defined in `syscall/types_windows.go` |
| `github.com/pkg/xattr` (`src/fs/attr.go`) | Ships `xattr_unsupported.go`; returns `ENOTSUP` |

**`syscall.Exec` being a silent runtime failure is worse than a compile error**, not better.
`plz run`, `plz tool`, `plz update`, `plz op` and the shim will build, ship, and then fail
with an opaque *"not supported by windows"*. The compiler cannot drive this work — it needs
tests. Likewise, narrowing the signal set and defaulting `Build.Xattrs = false` are
*correctness* changes with no build-time signal.

### Compiles, behaves wrong

Covered in M2. `PATH` split on literal `":"` (7 sites in `src/core/config.go` and
`src/core/utils.go`, plus `src/remote/action.go`); `src/fs/home.go` reading `$HOME`
directly; `/etc/please/plzconfig` and `DefaultPath`; `const SandboxDir = "/tmp/plz_sandbox"`;
the executable-bit model; `os.Symlink` privileges; `RemoveAll`'s chmod-to-force-delete;
hardcoded `sh -c` in `src/cache/cmd_cache.go`; `HOME=tmpDir` in `src/core/build_env.go`.

One of these was **confirmed as a hard startup blocker**, not a cosmetic issue:
`src/cli/logging.go:64` uses `path.Dir(logFile)` where it needs `filepath.Dir`, so `plz`
cannot create its log directory and dies before doing anything. Fix it early in M1, not in M2
— nothing can be tested under Wine until it is fixed.

## Design: process control via Job Objects

`src/process` is the highest-leverage package — every build action, test and `plz run`
funnels through it. It is also where Unix and Windows differ most.

Three separate Unix mechanisms collapse into one Windows primitive:

| Unix | Where | Purpose |
|---|---|---|
| `SysProcAttr{Setpgid: true}` | `exec_other.go`, `exec_linux.go` | Group the child and its descendants |
| `Pdeathsig: syscall.SIGHUP` | `exec_linux.go` | Kill orphans if plz dies |
| `syscall.Kill(-pid, sig)` | `process.go` | Signal the whole group |

**On Windows all three are a Job Object** created with
`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. Assign the child to the job at creation; every
descendant inherits membership; closing the handle (including on abnormal plz exit) kills
the tree. `TerminateJobObject` is the group kill.

### Graceful-then-forceful termination

`killProcess` currently sends `SIGTERM`, waits 30ms, then `SIGKILL`, waits 1s. There is a
deliberate comment in `ExecWithTimeout` explaining why `exec.CommandContext` is *not* used:
it only sends `SIGKILL`, which children cannot handle. **Preserve that intent.**

The Windows equivalent:

1. `GenerateConsoleCtrlEvent(CTRL_BREAK_EVENT, pgid)` — the closest thing to `SIGTERM`.
   Requires the child to have been created with `CREATE_NEW_PROCESS_GROUP`.
2. Wait the same 30ms.
3. `TerminateJobObject` — the `SIGKILL` equivalent.

Ctrl-Break delivery is unreliable for GUI subsystem processes and for children that have
detached from the console. Treat step 1 as best-effort; step 3 is the guarantee.

### Where this lands

```
src/process/exec_windows.go     // ExecCommand: create job, CREATE_NEW_PROCESS_GROUP
src/process/kill_windows.go     // killProcess, sendSignal
src/process/kill_unix.go        // //go:build !windows — the existing signal path
```

`process.go` keeps the timing policy and the executor bookkeeping; only the two primitives
(`sendSignal`, group kill) move behind the build tag.

## Design: `process.ExecReplace`

`syscall.Exec` replaces the current process image. Windows has no equivalent — Go's
`syscall/exec_windows.go` defines it as a stub returning `EWINDOWS`.

**None of the six call sites break compilation.** They build, ship, and fail at runtime with
an opaque error. That makes this the highest-risk item in M1: there is no compiler signal, so
it must be driven by tests.

Introduce one helper rather than six ad-hoc fixes:

```go
// ExecReplace replaces the current process with the given command where the OS supports
// it, and otherwise runs it as a child and exits with its status. It does not return on
// success.
func ExecReplace(argv []string, env []string) error
```

- **Unix** (`exec_replace_unix.go`): `syscall.Exec(argv[0], argv, env)`. Behaviour unchanged.
- **Windows** (`exec_replace_windows.go`): spawn, wait, `os.Exit(child.ExitCode())`.

### The behavioural difference, and why it matters

On Windows `plz run` and `plz tool` become a *parent process that outlives the child*. That
is not a transparent substitution, and three things follow:

1. **Signal forwarding must be explicit.** Ctrl-C in the console reaches both processes; the
   parent must not exit before the child has finished cleaning up, or the user sees plz's
   exit code instead of the program's.
2. **The parent must not hold the repo lock** while waiting. `src/core/lock.go` writes the
   PID into `plz-out/.lock`; a parent blocked in `Wait()` holding an exclusive lock
   deadlocks any nested plz invocation. Release before spawning.
3. **Exit codes must round-trip exactly.** `ExitError.ExitCode()` on Windows returns the
   raw process exit code, which for a crashing program is an `NTSTATUS` (e.g.
   `0xC0000005`). Do not truncate it to 8 bits.

Document all three in the code comment on `ExecReplace`, not just here.

### Special case: `tools/please_shim`

The shim's entire design is exec-replace: resolve `~/.please`, check the version, hand off.
On Windows it additionally needs `.exe`-aware path construction —
`filepath.Join(Location, "please")` must become `please.exe`. It is a separate binary with
its own `main`, so it needs its own copy of the helper or a shared package.

## Design: file locking

`src/core/lock.go` opens `plz-out/.lock` and holds an advisory `flock` on it, reusing one
file descriptor so the lock mode can be upgraded/downgraded in place. The file header says
so explicitly: *"The logic below relies heavily on flock (advisory locks)."*

Split into:

```
src/core/lock_unix.go      // //go:build !windows — syscall.Flock, unchanged
src/core/lock_windows.go   // LockFileEx / UnlockFileEx
```

Mapping:

| Unix | Windows |
|---|---|
| `LOCK_SH` | `LockFileEx` with no flags |
| `LOCK_EX` | `LockFileEx` with `LOCKFILE_EXCLUSIVE_LOCK` |
| `LOCK_NB` | `LOCKFILE_FAIL_IMMEDIATELY` |
| `LOCK_UN` | `UnlockFileEx` |

Two semantic differences to handle:

- **`LockFileEx` locks a byte range, not the file.** Lock `[0, 1)` consistently; the code
  writes a PID into the file, so do not lock a range that the write touches, or use a
  distinct offset well past any content.
- **Mode upgrade is not atomic.** `flock` can atomically convert shared → exclusive on the
  same fd. On Windows you must unlock and relock, which opens a race window. The existing
  callers (`AcquireSharedRepoLock` / `AcquireExclusiveRepoLock`) acquire once at startup, so
  this is tolerable — but assert it rather than assuming it, and cover it in
  `lock_test.go`.

This is the most self-contained piece of M1 and the best place to start.

## Design: signal handling

`src/cli/process.go` registers `SIGHUP`, `SIGINT`, `SIGQUIT`, `SIGABRT`, `SIGTERM` and exits
with `128 + signum`.

On Windows, Go's `signal` package delivers only `os.Interrupt` (Ctrl-C) and
`syscall.SIGTERM` (synthesised). The rest are unusable. Narrow the set behind a build tag,
and note that the `128 + signum` convention is a shell idiom with no meaning on Windows —
exit `1` instead. The `AtExit` handler machinery itself is portable and stays shared.

## Design: xattrs

`src/fs/attr.go` uses `github.com/pkg/xattr` to store content hashes as extended attributes,
consumed by `src/test/test_step.go` (`user.plz_test`) and `src/build`.

**A fallback already exists and is well-factored.** `RecordAttr` takes an `xattrsEnabled
bool` and delegates to `RecordAttrFile` (a sidecar file) when false. The config knob is
`config.Build.Xattrs`.

So the work is small:

1. Default `Build.Xattrs` to `false` on Windows.
2. Confirm `pkg/xattr` compiles for `GOOS=windows`. It ships a stub returning `ENOTSUP`, in
   which case no build tag is needed at all and step 1 is sufficient. **Verify this in M0
   rather than assuming it** — if the stub is absent, split `attr.go` into
   `attr_unix.go`/`attr_windows.go`.

Note the chmod-to-set-xattr dance in `RecordAttr` (chmod `|0200`, set, restore) becomes
dead code on Windows, which is fine — it is behind the `xattrsEnabled` branch.

## Design: the `.exe` model

The question with the widest blast radius, and the one most likely to be over-engineered.

Today, executability is a mode bit: `core.BuildTarget.OutMode()` returns `0555` for binary
targets and `0444` otherwise, applied by `src/build/build_step.go`. `src/fs/executable.go`
checks `(mode & 0111) == 0`. Nothing anywhere handles `.exe`.

**Recommendation: do not change the core model.**

`OutMode()` stays as-is — the mode bits are simply ignored by Windows, which is harmless.
The `.exe` suffix becomes a *rule-level* concern, handled in cc-rules (M5) and the go
plugin (M8) where the output name is chosen. This keeps the `plz-out` layout identical
across platforms and avoids threading a platform flag through `BuildTarget`.

Core only needs `.exe` awareness in the three places where it *looks up* an executable
rather than declaring one:

- `src/fs/executable.go` — the `mode & 0111` check and the `$PATH` search in `Executable()`.
  Use `PATHEXT` on Windows.
- `src/run/run_step.go` — `!strings.Contains(args[0], "/")` decides "is this a bare command
  name". Needs to consider `\` too.
- `tools/please_shim/main.go` — `filepath.Join(Location, "please")`.

If this turns out wrong — specifically, if `plz run` on a `cc_binary` cannot find its output
without core knowing about `.exe` — revisit before M6 rather than patching around it.

## Reuse rather than reinvent

Already in the tree, correct, and currently under-used:

- **`src/fs/executable.go` `splitPathList`** — a correct `os.PathListSeparator`-based split,
  used only by the FreeBSD `Executable()` fallback. Promote it to exported
  `fs.SplitPathList`/`fs.JoinPathList` and use it for all 7 raw `":"` splits in M2. Do not
  write a new one.
- **`rules/misc_rules.build_defs` `is_platform`** — the platform conditional for BUILD
  files. `src/BUILD.plz` uses it for the Linux-only `ldd` static-link assertion; that is the
  pattern to copy for Windows-conditional packaging.
- **`src/core/state.go` `ForArch`** — per-arch config layering (`.plzconfig_<os>_<arch>`).
  No changes needed; `.plzconfig_windows_amd64` slots straight in.
- **`config.Build.Xattrs`** — the xattr fallback, above.
- **`Remote.Shell`** (`src/core/config.go`, `src/remote/remote.go`) — the only configurable
  shell in the codebase today, and the precedent for the local `[build] Shell` knob in
  `02-shell-and-build-actions.md`.
- **`cli.Arch`** (`src/cli/flags.go`) — generic `OS_ARCH` parsing. `windows_amd64` parses
  today with no code change.
