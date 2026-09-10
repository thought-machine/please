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
| M1 | OS abstraction layer | 1–2w | ⬜ | — | — |
| M2 | Paths, environment and the `.exe` model | 1w | ⬜ | — | — |
| M3 | Build actions and the bundled shell | 1w | ⬜ | — | — |
| M4 | Release pipeline: cross-built Windows artifacts | 1w | ⬜ | — | — |
| M5 | C++ on Windows: cc-rules (workstream B) | 2w | ⬜ | — | — |
| M6 | Linux-hosted verification harness | 1w | ⬜ | — | — |
| M7 | Sandboxing parity | 2w | ⬜ | — | — |
| M8 | Remote execution and plugin parity | 3w | ⬜ | — | — |
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
- [ ] Non-blocking CI job: `plz build --arch windows_amd64 //src:please`
- [ ] `go1.27.0.windows-amd64` hash in `third_party/go/BUILD` (note: `.zip`, not `.tar.gz` —
      confirm `go_toolchain` handles it)
- [ ] `tools/images/windows_builder/Dockerfile`, added to `tools/images/build.sh`

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
BUILD-file path (not raw `go build`), and `//src/...` unit tests compile.

Design: `01-os-abstraction.md`. `probe/m1-skeleton.patch` is a starting shape — but its
`lock_windows.go` and `kill_windows.go` are deliberately wrong and must be replaced, not
adopted.

**The compile fixes are ~2 days. The rest of the milestone is the runtime work the compiler
gives no signal for.** Do the cheap fixes first to unblock Wine testing, then the real ones.

- [ ] **First, to unblock Wine testing:** `src/cli/logging.go` `path.Dir` → `filepath.Dir`
      (hard startup blocker), and `-tags forceposix` on `//src:please` + `//tools/please_shim`
- [ ] `src/output/shell_output.go` — remove the `SysProcAttr` leak; expose intent from
      `src/process` instead
- [ ] `src/core/lock.go` → `lock_unix.go` / `lock_windows.go` (`LockFileEx`) — **real
      implementation**, the probe's no-op would corrupt concurrent builds
- [ ] `process.ExecReplace` helper + 6 call sites — **no compiler signal; write tests first**
- [ ] `src/process/exec_windows.go` — Job Objects, `CREATE_NEW_PROCESS_GROUP`
- [ ] `src/process/kill_windows.go` / `kill_unix.go` — Ctrl-Break then `TerminateJobObject`
- [ ] Narrow `exec_other.go` from `!linux` to `!linux && !windows`
- [ ] `src/clean/clean.go` — `ForkExec` → detached `exec.Command`
- [ ] `src/cli/process.go` — narrow the signal set
- [ ] `src/fs/attr.go` — default `Build.Xattrs = false` on Windows (no build tag needed)
- [ ] `src/fs/executable.go` — `.exe` / `PATHEXT`
- [ ] `src/run/run_step.go` — `ExitError.ExitCode()` instead of `syscall.WaitStatus`

## M2 — Paths, environment and the `.exe` model

**Exit:** `//src/core/...` and `//src/fs/...` unit tests pass under Wine, including
`lock_test.go` and config loading.

Design: `01-os-abstraction.md` (the `.exe` model) and `02-shell-and-build-actions.md` (the
path-format rule).

- [ ] Promote `splitPathList` → `fs.SplitPathList`/`fs.JoinPathList`; replace 7 raw `":"`
      splits in `src/core/config.go`, `src/core/utils.go`, `src/remote/action.go`
- [ ] `src/fs/home.go` — `os.UserHomeDir()`; rework the `~` regex
- [ ] `src/core/config.go` — platform-conditional `MachineConfigFileName`, `DefaultPath`
- [ ] `src/core/build_env.go` — `USERPROFILE`, `TEMP`/`TMP`
- [ ] `src/core/build_target.go` — platform-conditional `SandboxDir`
- [ ] `src/fs/copy.go` — symlink privilege fallback
- [ ] `src/fs/fs.go` — `RemoveAll` clears `FILE_ATTRIBUTE_READONLY`
- [ ] Bug fixes: raw `"/"` splits in `src/fs/sort.go`, `src/fs/glob.go`,
      `src/build/build_step.go` (the `src/cli/logging.go` `path.Dir` fix moved to M1 — it
      blocks startup entirely)
- [ ] Forward-slash normalisation in `BuildEnvironment` + a test asserting no `\`

## M3 — Build actions and the bundled shell

**Exit:** a `genrule` with `cmd = "cat $SRCS | sort > $OUT"` builds under `plz.exe` on Wine.
*Already demonstrated in M0 with a hand-placed `bash.exe`; this milestone is about doing it
through config and packaging rather than by hand.*

Design: `02-shell-and-build-actions.md`.

- [ ] `[build] Shell` / `ShellArgs` config
- [ ] `src/process/process.go`, `src/run/run_step.go` — use it instead of literal `"bash"`
- [ ] `src/cache/cmd_cache.go` — replace hardcoded `sh -c` (2 sites)
- [ ] Vendor `busybox.exe` (`remote_file`, pinned hash, GPL-2.0 noted)
- [ ] Add to `//package:installed_files` under `is_platform(os = "windows")`
- [x] Applet and flag audit against busybox-w64 — done in M0, see
      `02-shell-and-build-actions.md`
- [ ] Gate the `xz -zc` tarball rule to Linux (busybox `xz` is decompress-only)
- [ ] `plz hash //...` unchanged on Linux

## M4 — Release pipeline

**Exit:** `plz build --arch windows_amd64 //package:release_files` on Linux CI produces a
signed `windows_amd64/` folder.

Design: `04-release-and-ci.md`.

- [ ] `src/parse/internal_package.go` — `windows_amd64` arcat hash **(hard gate)**
- [ ] `.plzconfig_windows_amd64`
- [ ] `package/BUILD` — gate `please_sandbox` on `is_platform(os = "linux")`
- [ ] `package/BUILD` — `.zip` release target
- [ ] `plz.cmd` shim instead of the `ln -sf please plz` symlink
- [ ] `src/update/update.go` — cannot overwrite a running `.exe`; use the version-directory
      layout
- [ ] `pleasew.ps1` + `src/assets/BUILD` + root `BUILD`
- [ ] `.circleci/config.yml` — `build-windows` job, workflow entry, `release-gs` requires
- [ ] `.circleci/release.sh` — `release_folder … windows_amd64/$VERSION`
- [ ] `tools/misc/gen_release.py` — `_arch()` windows branch

## M5 — C++ on Windows (workstream B)

**Exit:** `plz build --arch windows_amd64 //test/...` in cc-rules produces PE32+ `.exe` and
`.dll`.

Design: `03-cc-toolchain.md`. Repo: `please-build/cc-rules`.

- [ ] **First:** verify `x86_64-w64-mingw32-g++ -v -Wl,-v` matches the existing GCC and GNU ld
      regexes in `cctool/tool.go`. D1 rests on this.
- [ ] `build_defs/arch.build_defs` — add `windows_amd64`
- [ ] `cc_binary` / `cc_test` → `.exe`; `cc_shared_object` → `.dll` + import library
- [ ] Verify `plz run //some:cc_binary` still resolves the renamed output
- [ ] Flag review: drop `-fPIC` and `-Wl,--build-id=none` for Windows
- [ ] `DefaultLdFlags` override — `-lpthread -ldl` are both wrong on MinGW
- [ ] `please_cc` `execvp_windows.go` (needed for native Windows, not for Axis 2)
- [ ] Parse-time error when `pkg_config_libs` is used on Windows
- [ ] MinGW cross-compile job in `plugin_test_cc.yaml`
- [ ] Upstream PR; bump `plugins/BUILD` revision

## M6 — Linux-hosted verification harness

**Exit:** one CI job builds and runs a C++ `cc_test` for Windows, from Linux, end to end.

Design: `05-testing-strategy.md`.

- [ ] Wine test macro for cross-compiled Go test binaries
- [ ] Wine CI job — `//src/core/...`, `//src/fs/...`, `//src/process/...`
- [ ] The genrule shell smoke test
- [ ] The headline end-to-end: `wine plz.exe` + MinGW + `cc_test`
- [ ] Make the Wine job blocking

## M7 — Sandboxing parity

- [ ] Default `Sandbox.Build`/`Sandbox.Test` false on Windows, with a clear log line
- [ ] `sandbox_windows.go` — Job Objects (reuse M1), restricted token, scrubbed environment
- [ ] Document the filesystem-isolation gap: no mount-namespace analogue; Windows Containers
      rejected as too large a dependency

## M8 — Remote execution and plugin parity

- [ ] `src/remote/action.go` `translateOS` — add `windows`
- [ ] go plugin — `windows_amd64` arch, `.exe` naming
- [ ] shell plugin — `sh_binary` needs a `.cmd`/busybox shim instead of `#!`
- [ ] python plugin — pex on Windows (prior art: ChangeLog #947)
- [ ] `src/watch` — document fsnotify's Windows limits

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
| A dropped `forceposix` tag silently breaks every label | Total CLI breakage, only visible at runtime | Add a Wine smoke test asserting `query alltargets //...` works |
| Backslash escaping in shell command strings | Intermittent, hard-to-diagnose build failures | The forward-slash rule in `02-shell-and-build-actions.md`, plus an assertion test |
| `.exe` needs to be a core concept after all | Rework of the M2 decision | Verify `plz run` on a `cc_binary` early in M5, before the rest of M5 depends on it |
| Hash drift invalidates every user's cache | Silent, affects all platforms | `plz hash //...` diff on every M1–M3 PR |
| `ERROR_SHARING_VIOLATION` on real Windows | Invisible until M9 | Listed explicitly in the M9 issue; design `RemoveAll` and the updater defensively now |
| arcat platform gate forgotten | `plz.exe` cannot parse anything, discovered late | Called out as a hard gate in M4; it fails at runtime on Windows, not at build time on Linux |
