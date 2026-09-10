# Appendix — Baseline Compile and Runtime Errors

Status: **Measured** · Milestone: M0 · Last updated: 2026-09-10

Real results, not predictions. Measured against Go 1.27.0 (`GOOS=windows GOARCH=amd64`) at
commit `8cddc25` (Release 17.33.0), with a Linux control build as a baseline.

**Headline: the port is far closer than the source survey suggested.** `please.exe` compiles,
links, parses BUILD files and executes build actions after ~290 lines of change. The hard
work is not making it build — it is making it correct.

## Method

```bash
GOOS=windows GOARCH=amd64 go build ./src/... ./tools/...
```

**This is misleading on its own.** Go stops at the first failing package in the dependency
graph, and `src/process` is a dependency of nearly everything. The first run reports 3 errors
in 1 package; 30 of 51 packages simply never get type-checked.

The errors are **layered, not parallel**. You cannot enumerate them up front — each fix
reveals the next layer. Build each package independently and iterate:

```bash
go list ./src/... ./tools/... > pkgs.txt
while read -r p; do GOOS=windows GOARCH=amd64 go build -o /dev/null "$p"; done < pkgs.txt
```

Plan M1 as an iterative loop, not as a checklist derived from a single error dump.

## Compile blockers — the complete set

Four layers, five sites, four packages. That is all.

| Layer | Package | Site | Error |
|---|---|---|---|
| 1 | `src/process` | `exec_other.go:17,18` | `unknown field Setpgid / Foreground in syscall.SysProcAttr` |
| 1 | `src/process` | `process.go:206` | `undefined: syscall.Kill` |
| 2 | `src/core` | `lock.go` ×10 | `undefined: syscall.Flock`, `LOCK_SH/EX/UN/NB` |
| 3 | `src/clean` | `clean.go:96` | `undefined: syscall.ForkExec` |
| 3 | `src/output` | `shell_output.go:467` | `cmd.SysProcAttr.Setpgid undefined` |

After layer 3, **every package compiles and `./src` links to a valid PE32+ binary.**

```
please.exe: PE32+ executable (console) x86-64, for MS Windows, 16 sections
```

### Layer 3 contains a site the source survey missed

`src/output/shell_output.go:467`:

```go
cmd := state.ProcessExecutor.ExecCommand(...)
// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
cmd.SysProcAttr.Setpgid = false
```

A caller **outside** `src/process` reaching into platform-specific process attributes. This is
an abstraction leak, and the fix is not a build tag here — it is to expose the intent from
`src/process` (e.g. `process.ClearProcessGroup(cmd)`, or a parameter on `ExecCommand`) so the
platform detail stays in one package. Worth auditing for other instances during M1.

## Corrections to the source survey

Predictions that were **wrong**, and why. All three arise from the same mistake: assuming
"Unix-only API" means "does not compile on Windows".

### `syscall.Exec` is not a compile blocker

Go's `syscall/exec_windows.go` defines:

```go
func Exec(argv0 string, argv []string, envv []string) (err error) {
	return EWINDOWS
}
```

It **compiles** and fails at **runtime**. All five call sites (`src/please.go`,
`src/run/run_step.go`, `src/tool/tool.go`, `src/update/update.go`,
`tools/please_shim/main.go`) build cleanly.

This is *more* dangerous, not less: `plz run`, `plz tool`, `plz update`, `plz op` and the shim
will build, ship, and then fail at runtime with an opaque *"not supported by windows"*. The
`process.ExecReplace` work in `01-os-abstraction.md` is still required — it just cannot be
driven by the compiler. It needs tests.

`syscall.Chdir` likewise exists on Windows.

### Signal constants are not a compile blocker

`syscall.SIGHUP`, `SIGQUIT` and `SIGABRT` are all defined in `syscall/types_windows.go`.
`src/cli/process.go` compiles unchanged. Narrowing the `signal.Notify` set is a *correctness*
change (Windows delivers only `os.Interrupt` and a synthesised `SIGTERM`), not a build fix.

### `pkg/xattr` needs no build tag

The module ships `xattr_unsupported.go`. It compiles for Windows and returns `ENOTSUP`.
The M0 open question is resolved: **defaulting `Build.Xattrs = false` on Windows is
sufficient.** No `attr_unix.go`/`attr_windows.go` split needed.

### `syscall.ForkExec` genuinely is absent

`src/clean` is the one prediction that held exactly.

## Runtime findings

Compiling is not the interesting part. These were found by running the binary under Wine 9.0
and are ordered as encountered — each one blocks everything after it.

### R1 — go-flags parses `//pkg:target` as a flag *(new; not in the original plan)*

```console
$ wine please.exe query alltargets //...
CRITICAL: unknown flag `/...'
```

`github.com/thought-machine/go-flags` ships `optstyle_windows.go`:

```go
// Windows uses a front slash for both short and long options.  Also it uses
// a colon for name/argument delimter.
const (
	defaultShortOptDelimiter = '/'
	defaultLongOptDelimiter  = "/"
	defaultNameArgDelimiter  = ':'
)
```

**This collides with Please's entire label syntax.** `//pkg:target` parses as option `/pkg`
with argument `target`; `//...` is an unknown flag. Every command taking a build label — which
is nearly all of them — is broken.

**Fix:** the file is guarded `// +build !forceposix`. Build with `-tags forceposix`:

```bash
go build -tags forceposix ./src
```

Verified: label parsing works completely with the tag. This is a one-line BUILD-file change
and must be recorded as a decision, because it is invisible in the source and will silently
regress if the tag is dropped. It also applies to `tools/please_shim` and any other go-flags
binary.

### R2 — `path.Dir` on a filesystem path blocks startup

```console
CRITICAL: Error opening log file: open Z:\...\plz-out\log\build.log: Path not found.
```

`src/cli/logging.go:64` uses `path.Dir(logFile)` instead of `filepath.Dir`. Predicted in
`01-os-abstraction.md` as a "genuine bug"; confirmed here as a **hard startup blocker**, not a
cosmetic issue. `plz` cannot run at all until it is fixed.

### R3 — no shell

```console
Error building target //:hello: exec: "bash": executable file not found in %PATH%
```

Exactly as designed for in M3. Everything upstream of the shell works.

## What works, verified end to end

With the four compile fixes, `-tags forceposix`, the `path.Dir` fix, and busybox-w64 on
`%PATH%` as `bash.exe`:

```console
$ wine please.exe query alltargets //...
//:hello

$ wine please.exe build //:hello
plz-out\gen\hello.txt

$ wine please.exe build //:pipeline //:findpipe
plz-out\gen\sorted.txt
plz-out\gen\found.txt
```

where `pipeline` is `cat $SRCS | sort > $OUT` and `findpipe` is
`find . -name '*.o' -or -name '*.a' | sort | tr '\n' ','` — the construct the cc rules depend
on. Both produce correct output.

**This means the parser (`src/parse/asp`), config loading, the build graph, target hashing,
`plz-out` population and build-action execution all already work on Windows.**

Note the predicted "hard gate" in `src/parse/internal_package.go` (the exhaustive arcat
platform switch) did **not** trigger for parse or for simple genrules. It is only reached when
the `_please` internal package is actually needed. Still required for M4, but it is not the
early blocker the plan implied.

## Reference artifact

The throwaway probe patch is at `probe/m1-skeleton.patch` (287 lines, 17 files).

**It is not an implementation.** `lock_windows.go` returns `nil` — a no-op lock — and
`kill_windows.go` kills only the direct child, not the tree. It exists to prove the layering
and to give M1 a starting shape. Do not ship it.

## Progress

| Date | Compile-blocking sites | Notes |
|---|---|---|
| 2026-09-10 | 5 (4 packages, 4 layers) | Baseline. `please.exe` links, parses and builds after ~290 lines of probe changes. |
