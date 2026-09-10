# Build Actions and the Bundled Shell

Status: **Draft** · Milestone: M3 · Last updated: 2026-09-10

ADR for decision **D2**: ship a POSIX shell inside the Windows release rather than depending
on one being installed. See `00-overview.md` for the decision summary.

## The problem

Every build action, test and `plz run --cmd` is a **shell string**, not an argv. It is
executed by `src/process/process.go`:

```go
func BashCommand(binary, command string, exitOnError bool) []string {
	if exitOnError {
		return []string{binary, "--noprofile", "--norc", "-e", "-u", "-o", "pipefail", "-c", command}
	}
	return []string{binary, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", command}
}
```

`binary` is the literal string `"bash"` at every local call site
(`process.go`, `src/run/run_step.go`). Only the *remote* execution path is configurable
(`Remote.Shell`, `src/core/config.go`).

The command strings are not incidentally shell-shaped — they are genuinely shell. The
built-in rules use `&&`, `>`, `echo`, `mkdir`, `cp -r`, `mv`, `xz`. The cc rules go much
further: backticks, `find`, `sort`, `sed`, `tr`, and `; R=$?; …; exit $R`.

The worst single line, from `cc-rules` `build_defs/cc.build_defs` (`_binary_build_flags`):

```sh
find . -name '*.o' -or -name '*.a' | sort \
  | sed -e 's/\(.*\)/"-Wl,-force_load","\1"/' | tr '\n' , | sed -e 's/.$//'
```

Windows has none of this.

## Options considered

| Option | Verdict |
|---|---|
| **Require MSYS2 / Git Bash on PATH** | Rejected. Every user needs an extra install, and MSYS2's `/c/foo` ⇄ `C:\foo` path translation is applied heuristically to arguments that look like paths — which silently mangles compiler flags. |
| **Rewrite cc rules to be shell-free** | Deferred, not rejected. Moving `find`/`sort`/whole-archive assembly into `please_cc` is architecturally cleaner and would benefit every platform. But it is a large change in a second repo, and blocking the Windows port on it inverts the priorities. Revisit after M6. |
| **Bundle busybox-w64** | **Chosen.** Pins behaviour, needs nothing installed, no path translation. Costs one vendored binary (~700KB) in the release. |

## Verification results

Two rounds: Linux BusyBox 1.36.1 first, then the real **busybox-w64 1.38.0-FRP-6075**
(<https://frippery.org/files/busybox/busybox64.exe>, SHA-256
`07bb1e5b095b00d68a695481f9240879f33c5724b40aa2308f999d54ed78f075`) under Wine 9.0.

**The two rounds disagreed, and the w64 result is the one that counts.** The caution
originally written here — *"re-run every check against the actual busybox.exe"* — was
load-bearing.

### busybox-w64 ships a `bash` applet

Better than expected. Its applet list includes `bash` as well as `sh` and `ash`, so Please's
hardcoded `"bash"` resolves if `busybox.exe` is copied or hardlinked to `bash.exe`. No
indirection needed for a first cut.

### Shell flags — `--noprofile`/`--norc` are rejected

Linux busybox tolerated bash's full flag set. **busybox-w64 does not:**

```console
$ wine bash.exe --noprofile --norc -e -u -o pipefail -c 'echo ok'
bash: bad option '--noprofile'
```

Everything else is accepted, with semantics identical to bash:

```console
$ wine bash.exe -e -u -o pipefail -c 'false; echo REACHED'   # exit=1, not reached
$ wine bash.exe -e -u -o pipefail -c 'echo "$NOPE"'          # NOPE: parameter not set, exit=2
$ wine bash.exe -u -o pipefail -c 'false | true'             # exit=1
```

**Consequence: `ShellArgs` is mandatory, not a hedge.** `BashCommand` must drop
`--noprofile --norc` on Windows. Those two flags exist to stop bash sourcing user rc files;
busybox's shell has no rc files to source, so dropping them loses no hermeticity.

### The cc-rules pipeline works verbatim

Under Wine, through `plz.exe`, as a real build action:

```python
genrule(
    name = "findpipe",
    outs = ["found.txt"],
    cmd = "mkdir -p d/e && touch d/a.o d/e/b.a && find . -name '*.o' -or -name '*.a' | sort | tr '\\n' ',' > $OUT",
)
```

produces `./d/a.o,./d/e/b.a,`. This is the construct `_binary_build_flags` depends on, and it
is the strongest evidence for D2.

`cat $SRCS | sort > $OUT` also produces correctly sorted output across two source files.

### Applet coverage

Present in busybox-w64 and used by Please's rules: `sh`, `ash`, `bash`, `find`, `sort`, `sed`,
`tr`, `cat`, `cp`, `mv`, `rm`, `mkdir`, `echo`, `printf`, `test`, `dirname`, `basename`,
`xargs`, `cut`, `which`, `env`, `tar`, `gzip`, `unzip`, `head`, `tail`, `wc`, `tee`, `touch`,
`ln`, `readlink`, `realpath`, `grep`, `awk`, `flock`, `install`, `make`.

**Gaps:**

1. **`pkg-config` is absent.** No applet, no shim. Documented as unsupported on Windows —
   see `03-cc-toolchain.md`.
2. **`zip` is absent** (only `unzip`). Relevant to the M4 `.zip` release target, which is
   produced on Linux, so not a problem.
3. **`xz` compression.** Linux busybox `xz` is decompress-only (`xz -zc` → `invalid option
   -- 'z'`). `rules/misc_rules.build_defs` uses `xz -zc -T 0 $SRCS > "$OUT"` for
   `tarball(xzip = True)`. Release artifacts are produced on Linux so this is not on the
   user's critical path — gate the rule on `is_platform(os = "linux")`. Re-verify against
   busybox-w64, whose applet list does include `xz`.

## Design

### Config

Add to `[build]`, mirroring the existing `Remote.Shell`:

```ini
[build]
Shell = bash            ; unix default
ShellArgs = --noprofile ; repeatable
ShellArgs = --norc
ShellArgs = -u
ShellArgs = -o
ShellArgs = pipefail
```

On Windows the default resolves to the bundled `bash.exe` (busybox) with
`ShellArgs = -u -o pipefail` — i.e. the same set **minus `--noprofile --norc`**, which
busybox-w64 rejects. The `-e` flag stays conditional on `target.ShouldExitOnError()` and is
appended by `BashCommand`, not configured.

Verified working shape (from the probe):

```go
func BashCommand(binary, command string, exitOnError bool) []string {
	argv := append([]string{binary}, shellArgs...)   // platform-specific
	if exitOnError {
		argv = append(argv, "-e")
	}
	return append(argv, "-u", "-o", "pipefail", "-c", command)
}
```

### Code changes

- `src/process/process.go` — `ExecWithTimeoutShellStdStreams` takes the shell from config
  instead of the literal `"bash"`. `BashCommand` gains an args parameter.
- `src/run/run_step.go` — same.
- `src/cache/cmd_cache.go` — replace hardcoded `exec.Command("sh", "-c", …)` (two sites)
  with the same knob.

### Packaging

Vendor `busybox.exe` via a `remote_file` with a pinned SHA-256, and add it to
`//package:installed_files` under `is_platform(os = "windows")`. The pattern to copy is the
Linux-only `ldd` assertion in `src/BUILD.plz`.

Pin an exact release. busybox-w64 is a third-party fork
(<https://frippery.org/busybox/>); record the source URL, version and hash in
`third_party/binary/BUILD` so the provenance is auditable, and note the licence (GPL-2.0)
in the release's licence file.

## The path-format rule

**This is the highest-risk detail in the milestone.** Get it wrong and failures will be
intermittent and baffling.

Build actions receive paths through the environment — `$TMP_DIR`, `$OUT`, `$OUTS`, `$SRCS`,
`$SRCS_<NAME>`, `$TOOLS_<NAME>` — assembled in `src/core/build_env.go`. Those values are
interpolated into a **shell string**, where `\` is an escape character. A Windows path like
`C:\plz-out\tmp\foo` becomes `C:plz-outtmpfoo` after one round of shell processing.

**The rule: Please uses forward slashes everywhere inside `plz-out` and everywhere in the
build environment, on every platform, including Windows.**

Three reasons:

1. **Win32 accepts forward slashes.** `CreateFileW` and the whole `Win32` file API treat `/`
   and `\` interchangeably. So does MinGW GCC. So does busybox.
2. **It keeps hashes identical across platforms.** Please's cache is content-hash based over
   the rule definition and environment. If `$OUT` is `a\b` on Windows and `a/b` on Linux,
   every target hash diverges — which is correct but wasteful, and makes cross-platform
   remote cache sharing impossible.
3. **It is the smaller change.** `filepath.Join` produces `\` on Windows, so the conversion
   point is well-defined: normalise on the way *into* the build environment
   (`BuildEnvironment`, `toolPath`) rather than auditing every producer.

**Exceptions**, which must be explicit and commented:

- Absolute paths with a drive letter (`C:/...`) are fine with forward slashes and should
  keep the drive letter.
- UNC paths (`\\server\share`) cannot be normalised. Detect and reject them as a repo root
  with a clear error rather than producing corrupt commands.
- Paths passed to Windows APIs directly (not through the shell) keep whatever
  `filepath` produces. Only the *build environment* is normalised.

Add a test in `src/core/build_env_test.go` asserting no `\` appears in any value returned by
`BuildEnvironment` on Windows.

### Related: `HOME` and `TMPDIR`

`src/core/build_env.go` sets `HOME=tmpDir` and `TMPDIR=tmpDir` for every action. On Windows,
tools look at `USERPROFILE` and `TEMP`/`TMP`. Set all of them (M2), pointing at the same
normalised tmp dir, so the hermetic-environment guarantee holds for Windows-native tools too.

## Exit criterion

```bash
# under Wine, with the bundled shell
wine plz-out/bin/windows_amd64/src/please.exe build //test/genrule:pipeline_test
```

where the target is a `genrule` with `cmd = "cat $SRCS | sort > $OUT"`. That exercises
argument interpolation, a pipe, a redirect and two applets in one action.
