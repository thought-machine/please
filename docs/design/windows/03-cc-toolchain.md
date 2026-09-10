# C/C++ Toolchain on Windows

Status: **Draft** · Milestone: M5 · Workstream B (`please-build/cc-rules`) · Last updated: 2026-09-10

ADR for decision **D1**: target MinGW-w64 GCC first, MSVC later. Building C++ projects is the
programme's driving use case, so this is the document that matters most.

## Where the rules live

**Not in this repo.** `plugins/BUILD` pins `please-build/cc-rules` at `v0.7.3`, fetched as a
plugin subrepo via `plugin_repo` (`rules/subrepo_rules.build_defs`), from
`https://github.com/please-build/cc-rules/archive/v0.7.3.zip`.

The only in-repo consumer of cc rules is `tools/sandbox/BUILD`.

There is also a **dead** `[Cpp]` config section in `src/core/config.go` (`CCTool`, `CppTool`,
`LdTool`, `ArTool`, …). It is a no-op in plz v17+ — the code even warns *"You're overriding
field %s which is deprecated in plz v17+"*. Ignore it; the live config is the plugin's
`[PluginConfig …]` block, addressed as `CONFIG.CC.*` in BUILD files and `-o plugin.cc:…` on
the command line.

## Decision: MinGW-w64 first

### Why not MSVC first

MSVC is what most real Windows C++ projects use, and it is the eventual target. But choosing
it first means taking on three unknowns simultaneously:

- a completely different flag dialect (`/c`, `/Fo`, `/EHsc`, `/link`, `.obj`, `.lib`),
- `vcvarsall.bat` environment discovery (INCLUDE/LIB/PATH, SDK version selection),
- and a Windows host, because `cl.exe` does not run on Linux.

That last point is disqualifying on its own. It would block every C++ change on the same
native-Windows CI that M9 exists to defer.

### Why MinGW

`x86_64-w64-mingw32-g++` **runs on Linux**. That is Axis 2 from `00-overview.md`: the entire
C++ codepath becomes testable on the development platform, producing real PE32+ binaries,
before any Windows machine exists.

And the flag surface is almost entirely reusable. `_binary_build_flags` in
`build_defs/cc.build_defs` already branches Apple-vs-GNU throughout, via `please_cc`'s
expression language:

```python
oflags += ["""'{{ !ld64 && !appleld ? ["-Wl,--start-group", "-Wl,--whole-archive"] }}'"""]
```

MinGW's `ld` is GNU ld. **It takes the GNU branch for free.** That is the entire payoff of
D1.

## `please_cc` — the extension point

The rules do not emit compiler command lines directly. They emit `please_cc` invocations:

```
"$TOOLS_PLEASE_CC" cc "$TOOLS_CC" -c -I . <flags> ${SRCS_SRCS}
```

`please_cc` (`tools/please_cc/`, ~1000 lines of Go) runs the compiler with `-v -Wl,-v`,
regex-matches the output to identify the compiler *and* the linker it will invoke, evaluates
any `{{ … }}` expressions in the arguments against that identity, and `exec`s the real tool.

Known identities today (`tools/please_cc/cctool/tool.go`): GCC, Clang, Apple Clang, GNU ld,
GNU gold, LLD, ld64, Apple ld.

### The assumption D1 rests on — **confirmed**

Measured against two toolchains, both by regex and in the real pipeline:

| Toolchain | Compiler line | Linker line | `please_cc` says |
|---|---|---|---|
| WinLibs GCC 16.2.0 (Windows-native, under Wine) | `gcc version 16.2.0 (MinGW-W64 …)` | `GNU ld (Binutils for MinGW-W64 …) 2.47.20260726` | GCC 16.2.0 / GNU ld 2.47.20260726 |
| Ubuntu `g++-mingw-w64-x86-64` 13 (Linux cross) | `gcc version 13-win32 (GCC)` | `GNU ld (GNU Binutils) 2.41.90.20240122` | GCC 13 / GNU ld 2.41.90 |

Both match the existing GCC and GNU ld matchers, and the Clang matcher correctly does not.
**No new matchers are needed.**

Note the Ubuntu build reports `13-win32`, so the captured version is a bare `13`.
`MustParseVersion` handles a single component, and `Compare` zero-pads the shorter of two
version numbers, so `gcc >= 9` style expressions still evaluate correctly.

The original check, kept for reference:

```bash
x86_64-w64-mingw32-g++ -v -Wl,-v 2>&1 | head -20
```

Check the output against these two patterns from `cctool/tool.go`:

```
^gcc (?:version|\(GCC\)) (?P<version>[\d.]+)
^GNU ld (?:\(.*\) |version )(?P<version>\d+(?:\.\d+)*)
```

If either fails to match, `please_cc` exits with *"failed to identify C/C++ compiler; please
report the output of … to <issues URL>"* and **every cc target fails**. If that happens, the
fix is small (add a matcher) but it must be known up front, not discovered mid-milestone.

Note the ordering constraint documented in `tool.go`: the Apple Clang matcher must run before
the Clang matcher because Go's `regexp` has no zero-length assertions. Any new matcher must be
placed with the same care.

### `please_cc` needs a Windows build

`tools/please_cc/please_cc.go` ends in:

```go
func execvp(file string, args []string) error {
	execFile, err := exec.LookPath(file)
	...
	return syscall.Exec(execFile, append([]string{file}, args...), os.Environ())
}
```

`syscall.Exec` does not exist on Windows. Add `execvp_windows.go` that spawns, waits and
propagates the exit code — the same shape as `process.ExecReplace` in `01-os-abstraction.md`.

**Priority note:** under Please's cross-compilation model, `tools` are always built for the
*host* arch. So for Axis 2 (cross-building C++ from Linux) `please_cc` runs as a Linux
binary and this is not on the critical path. It is only required for a native Windows `plz`.
Sequence it accordingly.

## Output extensions

Current naming, all in `build_defs/cc.build_defs`:

| Rule | Output | MinGW needs |
|---|---|---|
| `cc_object` | `<name>.o` | unchanged — MinGW uses `.o` |
| `cc_library` | `lib<name>.a` | unchanged — MinGW uses `.a` |
| `cc_static_library` | `lib<name>.a` | unchanged |
| `cc_shared_object` | `lib<name>.so` | **`<name>.dll`**, plus `lib<name>.dll.a` (import library) as an `optional_out` |
| `cc_binary` | `<name>` (bare) | **`<name>.exe`** |
| `cc_test` | `<name>` (bare) | **`<name>.exe`** |

This is the one place MinGW's GNU-ness does not carry over, and it is why D1 saves work
rather than eliminating it: only two of six naming schemes change.

Gate on `CONFIG.OS == "windows"`. The rules already precedent OS-conditional logic:

```python
if CONFIG.TARGET_OS == "darwin" and static:
    log.warning("%s: statically-linked binaries are unsupported on Darwin; ignoring 'static'")
    static = False
```

Also add `windows_amd64` to `SUPPORTED_ARCHITECTURES` in `build_defs/arch.build_defs`, which
today lists only the five released platforms.

### Consequence for `plz run` and `cc_test`

`cc_binary` sets `binary = True` and `outs = [name]`. Changing the out to `name + ".exe"`
changes the path `plz run` and the test runner resolve. Per `01-os-abstraction.md`, core is
*not* being taught about `.exe`, so verify early that `plz run //some:cc_binary` works with
the renamed output — this is the specific case that would force a rethink.

## Flag review

Every flag in `_build_flags` and `_binary_build_flags`, assessed for PE/COFF via MinGW.

| Flag | Where | Verdict |
|---|---|---|
| `-c`, `-I .` | `_library_cmds` | Fine |
| `-fPIC` | `_build_flags` | **Remove for Windows.** Meaningless for PE; MinGW emits *"-fPIC ignored for target"* on every compile. Noise, not breakage — but it pollutes every build log. |
| `-fdata-sections`, `-ffunction-sections` | `_build_flags` (LdGarbageCollection) | Fine — supported by MinGW GCC |
| `-fno-unique-section-names` | `_build_flags` | Clang-only branch already; no change |
| `-Wl,--start-group` / `--end-group` | `_binary_build_flags` | Fine — GNU ld branch, taken automatically |
| `-Wl,--whole-archive` / `--no-whole-archive` | `_binary_build_flags` | Fine — same |
| `-Wl,--build-id=none` | `_binary_build_flags` | **Remove for Windows.** ELF-only; MinGW ld errors or warns. Guard the existing `{{ gnuld \|\| gold \|\| lld ? … }}` expression with an OS check, since MinGW ld *does* match `gnuld`. |
| `-Wl,--gc-sections` | `_binary_build_flags` | Fine — MinGW ld supports it |
| `-Wl,--strip-all` | `_binary_build_flags` | Fine |
| `-shared` | `_binary_build_flags` | Fine — produces a DLL |
| `-static` | `cc_binary` | Works, but means "static libgcc/libstdc++" rather than a fully static image. Document the difference; do not silently disable it as Darwin does. |
| `-static-libgcc` | `cc_binary` (gcc branch) | Fine |
| `--coverage`, `-fprofile-dir=.` | `_COVERAGE_FLAGS` | Works with MinGW + `gcov`, but the `cover` command copies `.gcno` from `$GCNO_DIR` and shells out — verify end-to-end in M6 rather than assuming |
| `DefaultLdFlags = -lpthread -ldl` | `.plzconfig` | **Both wrong on MinGW.** `-ldl` does not exist; `-lpthread` is unnecessary (winpthreads is implicit) and may not resolve. Needs a `windows_amd64` override — set `DefaultLdFlags` to empty. |

The `-fPIC` and `-Wl,--build-id=none` changes both need a way to express "target OS is
windows" inside the `{{ … }}` expression language, or an `if CONFIG.OS == "windows"` in the
Python-side flag assembly. **Prefer the latter** — the expression language identifies
*tools*, not targets, and overloading it with OS knowledge would be a category error.

## `pkg-config`

`_build_flags` and `_binary_build_flags` emit backticked `pkg-config --cflags` /
`--libs` invocations. There is no `pkg-config` in busybox and no Windows convention for it.

**Decision: document as unsupported on Windows.** Leave the codepath intact — it will simply
fail if used — and have users pass `compiler_flags`/`linker_flags` explicitly. Do not ship a
`pkg-config` shim; that is a package-management problem, not a build-system one.

Emit a clear diagnostic rather than a shell "command not found": add a check in the rules
that raises at parse time when `pkg_config_libs` is set and `CONFIG.OS == "windows"`.

## What MSVC would later require

Recorded so the extension point stays visible, not as scheduled work.

1. **New `cctool` matchers.** `cl.exe /?` prints *"Microsoft (R) C/C++ Optimizing Compiler
   Version 19.NN.NNNNN"*; `link.exe` prints *"Microsoft (R) Incremental Linker Version …"*.
   Note that `please_cc` currently probes with `-v -Wl,-v`, which MSVC does not understand —
   the probe itself needs to become tool-family-aware, which is a deeper change than adding
   a regex.
2. **A second flag dialect.** Not a translation layer — a parallel set of flag-assembly
   functions selected by toolchain, because the mappings are not one-to-one
   (`--whole-archive` → `/WHOLEARCHIVE:lib`, `--gc-sections` → `/OPT:REF`, and
   `--start-group` has no equivalent at all because MSVC's linker does not care about
   library order).
3. **Separate archiver and linker tools.** MinGW links through the compiler driver
   (`$TOOLS_CC`); MSVC needs `lib.exe` and `link.exe` as distinct tools. The plugin config has
   no `LdTool` today — it would need one.
4. **`vcvarsall` environment discovery.** `INCLUDE`, `LIB`, `LIBPATH` and SDK version
   selection. This conflicts with Please's hermetic-environment model
   (`src/core/build_env.go` builds the env from scratch rather than inheriting), so it needs
   a deliberate design — most likely a `pass_env` allowlist plus a documented setup step.
5. **`.obj`/`.lib`/`.pdb`** output naming, and `/showIncludes` if header scanning is ever
   added (it is not today — see below).

`clang-cl` is a middle path: one binary, MSVC-compatible flags, already partially matched by
the existing Clang regex. It still needs items 2, 4 and 5.

## What is pleasantly absent

**No header-dependency scanning.** There is no `-MD`, `-MF`, `-MMD` or `.d` handling
anywhere in the rules. Please does not scan headers; correctness comes from declared
`hdrs`/`private_hdrs` plus the sandboxed tmp dir.

This removes a whole class of portability work — no dep-file path munging, no
`/showIncludes` parsing when MSVC eventually lands.

## Development and upstreaming

Point `plugins/BUILD` at a fork or branch revision during development:

```python
plugin_repo(
    name = "cc",
    plugin = "cc-rules",
    revision = "<branch-or-sha>",
)
```

Upstream to `please-build/cc-rules` as the final step of M5. Per `CONTRIBUTING.md`, raise the
issue in that repo *before* writing the code.

Extend the plugin's own CI (`.github/workflows/plugin_test_cc.yaml`) with a MinGW
cross-compile job on `ubuntu-latest` — `apt-get install g++-mingw-w64-x86-64` plus
`plz build --arch windows_amd64 //test/...`.

## Exit criterion — met

On a Linux box, in the cc-rules repo with `cc-rules-windows.patch` applied:

```console
$ plz build --arch windows_amd64 //test/binary:test_binary
plz-out/bin/windows_amd64/test/binary/test_binary.exe
$ file plz-out/bin/windows_amd64/test/binary/test_binary.exe
PE32+ executable (console) x86-64, for MS Windows
```

A `cc_library` + `cc_binary` + `cc_shared_object` triple produces `lib.a`, `prog.exe` and
`libshared.dll`; `prog.exe` links against the static library and prints the right answer under
Wine. The same targets still produce `prog` and `libshared.so` on Linux, and all 12 of
cc-rules' own tests pass there.

### What the experiments changed

1. **Module-level `CONFIG` does not see the target architecture.** The first attempt defined
   `_EXE_SUFFIX` as a module-level constant and it silently had no effect — these build defs
   are subincluded, and `CONFIG.OS` at module level reflects the host. It has to be a function
   evaluated per call. This is a trap for any future platform-conditional logic here.
2. **A repeatable config key cannot be cleared by assigning empty.** `defaultldflags =`
   yields a list containing one empty string rather than an empty list, which
   `_escape_linker_flag` turns into a bare `-Wl,` and the linker rejects with
   `cannot find : Invalid argument`. Set an actual value instead.
3. **`-lpthread` is fine on MinGW**, so only `-ldl` had to go. The Windows default is
   `defaultldflags = -lpthread`.
4. **`-fPIC` and `-Wl,--build-id=none` were passed and neither broke the link.** They remain
   worth removing as noise, but they are not blockers, so that is deferred rather than done.
5. **A `cc_shared_object` that sets `out` explicitly keeps whatever extension it was given.**
   The rules' *default* is now correct, but a BUILD file hardcoding `out = "libfoo.so"` — as
   cc-rules' own `//test/so:libdolphin` does — will still produce a `.so` on Windows. That is
   arguably right, since `out` is an explicit instruction, but it is a portability trap worth
   documenting for users.

### Still open

- **`please_cc` has no `windows_amd64` release.** `tools/BUILD` fetches it as a prebuilt
  binary per platform with a pinned hash, so upstreaming needs a Windows build published
  alongside the others. It did not block this work because tools are built for the *host*,
  which is Linux under Axis 2 — but a native Windows `plz` will need it.
- **`UnitTest++` does not compile for Windows** as packaged: it needs its `Win32/` platform
  sources, which the plugin's target does not include. This blocks `cc_test`, not
  `cc_library`/`cc_binary`.
- `SUPPORTED_ARCHITECTURES` still lacks `windows_amd64`; it gates the plugin's own release
  rather than its use.
