# Release Pipeline and CI

Status: **Draft** · Milestone: M4 · Last updated: 2026-09-10

How `windows_amd64` artifacts get built on Linux, signed and published. This is Axis 1 from
`00-overview.md`.

## The template: FreeBSD

Please already cross-compiles a platform it does not test natively on CircleCI. The
`build-freebsd` job runs **on Linux**, in a Docker image, using a prebuilt Linux `plz`:

```yaml
build-freebsd:
  docker:
    - image: ghcr.io/thought-machine/please_freebsd_builder:20260318
  steps:
    - checkout
    - attach_workspace: { at: /tmp/workspace }
    - run:
        name: Extract plz
        command: tar -xzf /tmp/workspace/linux_amd64/please_*.tar.gz
    - run:
        name: Cross-compile
        command: ./please/please build -p -v2 --profile ci --arch freebsd_amd64 //package:release_files
    - persist_to_workspace:
        root: plz-out/pkg
        paths: [ freebsd_amd64/* ]
```

`build-windows` is a near-verbatim copy. **Copy it; do not invent a new shape.**

Note the dependency: `build-freebsd` `requires: [build-alpine]`, because `build-alpine`
produces the canonical `linux_amd64` release tarball that every cross job extracts and runs.
`build-windows` takes the same dependency.

The builder image is equally simple — `tools/images/freebsd_builder/Dockerfile` is 13 lines:

```dockerfile
FROM ubuntu:noble
RUN apt-get update && apt-get install -y curl git gcc xz-utils && apt-get clean
RUN curl -fsSL https://dl.google.com/go/go1.26.1.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
RUN GOOS=freebsd go install std
```

`tools/images/windows_builder/Dockerfile` is the same with `GOOS=windows go install std`,
plus `g++-mingw-w64-x86-64` in the apt line for Axis 2. Add the image name to the list in
`tools/images/build.sh`; images are tagged by date and pushed to
`ghcr.io/thought-machine/please_<image>`.

## Prerequisites

Four things must land before the cross-build can even start.

### 1. Go toolchain hash

`third_party/go/BUILD` pins per-platform SHA-256s for the Go distribution:

```python
go_toolchain(
    name = "toolchain",
    hashes = [
        "…",  # go1.27.0.darwin-amd64.tar.gz
        …
    ],
    version = "1.27.0",
)
```

Add `go1.27.0.windows-amd64.zip`. Note Windows Go distributions are `.zip`, not `.tar.gz` —
confirm the go plugin's `go_toolchain` rule handles that, or the hash is useless.

### 2. arcat platform gate

`src/parse/internal_package.go` has an exhaustive switch that **hard-fails** on unknown
platforms:

```go
default:
    return "", fmt.Errorf("arcat tool not supported for platform: %s_%s", runtime.GOOS, runtime.GOARCH)
```

Without a `windows_amd64` entry, `plz.exe` cannot parse a single BUILD file. This is the
hardest gate in the whole milestone and the easiest to overlook, because it fails at
*runtime* on Windows, not at build time on Linux.

The hash is of `please_tools_<version>.tar.xz` for the platform, which is itself produced by
`//package:please_tools_tarball` — so it is a chicken-and-egg step: build the tools tarball
for windows once, record its hash, commit it.

### 3. `.plzconfig_windows_amd64`

Per-arch config, layered by `state.ForArch` (`src/core/state.go`). Compare
`.plzconfig_freebsd_amd64`, which is two lines.

```ini
[Plugin "cc"]
cctool = x86_64-w64-mingw32-gcc
cpptool = x86_64-w64-mingw32-g++
artool = x86_64-w64-mingw32-ar
defaultldflags =        ; -lpthread -ldl are both wrong on MinGW

[build]
xattrs = false

[sandbox]
build = false
test = false
```

### 4. `//package:installed_files` must stop pulling in the Linux sandbox

```python
filegroup(
    name = "tools",
    srcs = [
        "//tools/build_langserver",
        "//tools/sandbox:please_sandbox",   # <- C binary of Linux-namespace code
    ],
)
```

`please_sandbox` is a `c_binary` whose source is `#ifdef __linux__` throughout, with a no-op
fallback. It has no meaning on Windows and building it requires a C toolchain for the target.
Gate it with `is_platform(os = "linux")` (`rules/misc_rules.build_defs`; see `src/BUILD.plz`
for the usage pattern).

**This is already latently wrong for the FreeBSD cross build**, which gets away with it
because `.plzconfig_freebsd_amd64` points `cctool` at the host Linux `cc`. Fixing it properly
benefits both platforms.

## Packaging

### Add a `.zip` alongside the tarballs

`package/BUILD` produces `please_<VERSION>.tar.gz`, `.tar.xz` and a tools tarball. Windows
has no guaranteed `tar -xJ`; ship a zip.

Note from `02-shell-and-build-actions.md`: busybox has `unzip` but not `zip`, and its `xz` is
decompress-only. Both are irrelevant here because release artifacts are *produced* on Linux —
but it does mean the xz tarball rule should be gated to Linux rather than attempted on
Windows.

### Contents of the Windows release

| File | Source |
|---|---|
| `please.exe` | `//src:please` |
| `busybox.exe` | vendored `remote_file`, pinned hash — see `02-shell-and-build-actions.md` |
| `build_langserver.exe` | `//tools/build_langserver` |
| — | **no** `please_sandbox` |

`//package:installed_files` sets `entry_points = {"please": "please"}` — verify this resolves
with the `.exe` suffix, or add a Windows-conditional entry point.

### The `plz` alias

`install.sh` does `ln -sf please plz`. On Windows, symlinks need Developer Mode. Ship a
`plz.cmd` one-liner (`@"%~dp0please.exe" %*`) instead — a file copy is also acceptable but
doubles the download size.

## Bootstrap and self-update

### `pleasew`

`pleasew` is POSIX `sh` and has an explicit OS whitelist:

```sh
Linux|Darwin|FreeBSD) ;;
*) echo "Please does not support the %s operating system"; exit 1 ;;
```

**Do not make it polyglot.** Add a sibling `pleasew.ps1` (PowerShell) implementing the same
flow: find repo root, read `.plzconfig`/`.plzconfig_<os>_<arch>` for the version, download
`${URL_BASE}/windows_amd64/${VERSION}/please_${VERSION}.zip`, extract, exec.

`pleasew` is embedded into the binary via `src/assets/BUILD` (`plz init` writes it out), so
`pleasew.ps1` needs adding there and to the root `BUILD` filegroup too.

### `src/update/update.go`

The download URL is already built from `runtime.GOOS`/`runtime.GOARCH`:

```go
url = fmt.Sprintf("%s/%s_%s/%s/please_%s%s", DownloadLocation, GOOS, GOARCH, Version, Version, ext)
```

so it works as soon as the bucket has a `windows_amd64/` folder. Three things around it do
not:

- `syscall.Exec(newPlease, …)` to hand over to the new binary → `process.ExecReplace`
  (`01-os-abstraction.md`).
- `writeTarFile` recreates `tar.TypeSymlink` members → needs the M2 copy-fallback.
- The binary is opened with mode `0555` and `fileMode()` returns `0664`/`0775` → harmless on
  Windows, but the symlink at the end (`please` → version dir) is not.

Also: **Windows will not let you overwrite a running executable.** The self-updater must
rename the running `please.exe` aside before writing the new one, or update into a
version-stamped directory and switch a `.cmd` shim. The version-directory layout Please
already uses (`~/.please/<version>/`) makes the second option natural.

### `tools/please_shim`

Same exec-replace problem, plus `filepath.Join(Location, "please")` needs `.exe`. Covered in
`01-os-abstraction.md`.

## CI wiring

`.circleci/config.yml`:

1. New `build-windows` job (copy `build-freebsd`), `requires: [build-alpine]`.
2. Add it to the workflow `jobs:` list.
3. Add it to `release-gs`'s `requires:` list alongside `build-freebsd`.

`.circleci/release.sh`:

```sh
release_folder /tmp/workspace/windows_amd64 windows_amd64/$VERSION
```

The signing globs above it are `{*_amd64,*_arm64}`, which **already match** `windows_amd64` —
so signing needs no change, but note that means an unreleased `windows_amd64` folder in the
workspace would be signed and then silently dropped. Add the `release_folder` line in the
same commit as the CI job, not later.

The idempotency guard at the top of `release.sh` checks whether
`gs://get.please.build/linux_arm64/$VERSION/` exists. Leave it — adding Windows to it would
make the first Windows release re-upload everything.

`tools/misc/gen_release.py` — its `_arch()` helper defaults anything non-darwin/non-freebsd
to `linux_*`. Add a windows branch, or the GitHub release assets get mislabelled.

## Non-blocking guardrail (M0)

Before any of the above, add a **non-blocking** job that runs:

```bash
plz build --arch windows_amd64 //src:please
```

It will fail. That is the point: it makes the compile-error count visible and
monotonically decreasing, and it catches regressions from contributors who are not thinking
about Windows. Record the initial output in `appendix-baseline-errors.md`.

## Exit criterion

```bash
plz build --arch windows_amd64 //package:release_files
ls plz-out/pkg/windows_amd64/
# please_<VERSION>.zip, please_<VERSION>.tar.gz, please_<VERSION>, please_shim_<VERSION>
```

on a Linux CI box, with the artifacts signed by the existing `release_signer` step.
