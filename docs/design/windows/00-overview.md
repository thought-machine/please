# Windows Port — Overview

Status: **Draft** · Owner: _unassigned_ · Last updated: 2026-09-10

This directory holds the engineering design documents for adding a Windows port to Please.
They are working documents for contributors, not user-facing documentation — the docs site
build (`docs/BUILD`) only globs `milestones/*.html` and does not pick this directory up.

Read this file first, then `06-milestones.md` for current status.

## Why

Please ships binaries for `linux_amd64`, `linux_arm64`, `darwin_amd64`, `darwin_arm64` and
`freebsd_amd64`. There is no Windows build. `pleasew` bails out with *"Please does not
support the %s operating system"*, and `src/cli/winch_windows.go` is the only Windows-aware
file in 272 Go source files.

The 15.9.1 milestone note said *"We're scoping out support for Windows"*. This is that
scoping, turned into a plan.

## Goal

`plz.exe` on Windows, at **full feature parity** with the Linux build, with **building C++
projects as the primary driving use case**.

Parity means sandboxing, remote execution and all four language plugins eventually work.
It does not mean they all arrive at once: the milestone sequence deliberately front-loads a
working C++ vertical slice, then fills in the rest.

## Constraint that shapes everything

**Development and CI stay on Linux.** The binary is cross-compiled. Real Windows testing is
a later stage (M9).

This is workable because of two independent cross-compilation axes. Conflating them is the
most likely way to lose a week, so they get separate names throughout these documents.

### Axis 1 — cross-building `plz.exe`

`GOOS=windows go build`, driven from a Linux host by:

```bash
plz build --arch windows_amd64 //package:release_files
```

Produces the Please binary we ship. Modelled on the existing FreeBSD release flow, which is
already a Linux-hosted cross build (`.circleci/config.yml`, the `build-freebsd` job).

### Axis 2 — cross-building C++ *for* Windows

`plz --arch windows_amd64` with `.plzconfig_windows_amd64` pointing `cpptool` at
`x86_64-w64-mingw32-g++`. Produces `.exe`/`.dll`/`.a` artifacts.

The important property: **this exercises every cc-rules Windows codepath while plz itself
runs on Linux.** Roughly 80% of the C++ work is verifiable before any Windows machine is
involved.

Layer Wine on top (M6) and `plz.exe` itself becomes testable on Linux too. See
`05-testing-strategy.md`.

## Decisions

Each decision has a fuller ADR in the linked document. D1–D4 were taken during planning;
D5 came out of the M0 investigation.

### D1 — MinGW-w64 GCC first, MSVC/clang-cl later

See `03-cc-toolchain.md`.

MinGW reuses the existing GNU-driver flag logic (`-Wl,--start-group`, `--whole-archive`,
`-Wl,--gc-sections`) and the existing `please_cc` GCC matcher essentially unchanged.
Critically, `x86_64-w64-mingw32-g++` runs **on Linux**, so Axis 2 works from day one.

MSVC would require a new flag dialect (`/c`, `/Fo`, `/EHsc`, `.obj`/`.lib`), `vcvarsall`
environment discovery, *and* a Windows host — three unknowns at once. It remains the
eventual target for most real-world Windows C++ projects, and `please_cc` is the designed
extension point for it.

### D2 — Bundle a POSIX shell (busybox-w64) in the Windows release

See `02-shell-and-build-actions.md`.

Build actions run through `bash --noprofile --norc -e -u -o pipefail -c`
(`src/process/process.go`), and the cc rules emit genuine shell pipelines — `find | sort |
sed | tr`, backticks, `&&`. Bundling busybox pins the behaviour, requires nothing installed
on the user's machine, and avoids MSYS2's `/c/foo` ⇄ `C:\foo` path translation.

The alternative considered and rejected: rewriting the cc rules to be shell-free. That is
architecturally cleaner and remains a good idea for its own sake, but it is a large change
in a second repo and would block the Windows port on it.

### D3 — Full feature parity is the destination

Sandboxing (M7), remote execution and the go/python/shell plugins (M8) are real milestones,
not a dropped backlog. The OS abstraction layer built in M1 is designed so they have
somewhere to land — in particular, the Job Object machinery introduced for process control
is also what a Windows sandbox will be built on.

### D4 — Two repos, one programme

The C/C++ rules are not in this repo. `plugins/BUILD` pins `please-build/cc-rules` at
`v0.7.3`, fetched as a plugin subrepo.

- **Workstream A** — `thought-machine/please`: the core port, the OS abstraction layer, the
  release pipeline.
- **Workstream B** — `please-build/cc-rules`: `please_cc`, output extensions, the MinGW
  flag review.

During development, point `plugins/BUILD` at a fork or branch revision. Upstream to
`please-build/cc-rules` as the final step of M5.

### D5 — Build every go-flags binary with `-tags forceposix`

Discovered by running the binary, not by reading the source. `go-flags` uses `/` as its option
delimiter and `:` as its name/argument delimiter on Windows, which collides with Please's
**entire label syntax**: `//pkg:target` parses as option `/pkg` with argument `target`, and
`//...` is rejected as an unknown flag.

The library guards that file with `// +build !forceposix`, so the fix is a build tag. Verified:
label parsing works completely with it, and is completely broken without it.

This must be recorded as a decision rather than a code comment, because it is invisible in
Please's own source and will silently regress if the tag is ever dropped. It applies to
`//src:please`, `tools/please_shim`, and any other go-flags binary.

See R1 in `appendix-baseline-errors.md`.

## Non-goals

Explicitly out of scope for this programme:

- **MSVC support.** Designed for (see `03-cc-toolchain.md`) but not built.
- **32-bit Windows.** `windows_amd64` only. `windows_arm64` is plausible later; nothing in
  the design precludes it.
- **`pkg-config` on Windows.** The codepath stays, but it is documented as unsupported.
  Users set flags explicitly.
- **Windows Containers for sandboxing.** M7 accepts the tmp-dir isolation Please already
  does and documents the gap rather than taking on that dependency.
- **Native Windows as a development platform.** M9 adds native CI; the day-to-day loop
  stays on Linux.

## Document index

| Document | Contents |
|---|---|
| `00-overview.md` | This file. Charter, decisions, axes, non-goals. |
| `01-os-abstraction.md` | The `_windows.go` convention, `ExecReplace`, Job Objects, file locking, xattrs. |
| `02-shell-and-build-actions.md` | ADR for D2. busybox applet audit. The path-format rule. |
| `03-cc-toolchain.md` | ADR for D1. MinGW flag matrix. Output extensions. The MSVC extension point. |
| `04-release-and-ci.md` | Cross-build and release pipeline, modelled on FreeBSD. |
| `05-testing-strategy.md` | MinGW for Axis 2, Wine for `plz.exe`, and what Wine misses. |
| `06-milestones.md` | The living tracker. Status, exit criteria, owners. |
| `appendix-baseline-errors.md` | **Measured** M0 results: compile blockers, runtime findings, what already works. |
| `probe/` | Throwaway M0 artifacts, incl. `m1-skeleton.patch`. Not implementations. |

## Process

Per `CONTRIBUTING.md`: raise a GitHub issue for each milestone **before** writing code, and
keep PRs small and single-purpose — no refactors mixed with features.

Two repo-specific hazards worth repeating here:

- **Hash stability.** Please's cache is content-hash based over rule definition, config,
  sources and secrets. Any change to command generation, environment variables or config
  defaults changes target hashes and invalidates every user's cache. Confirm `plz hash
  //...` is unchanged on Linux before merging anything in M1–M3.
- **Brittle e2e tests.** The tests in `test/` assert on exact output text. Expect to update
  `.txt` golden files; treat any *unexpected* change there as a real regression.
