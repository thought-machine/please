# Probe artifacts

Throwaway material from the M0 investigation. **Not implementations — do not ship.**

- `m1-skeleton.patch` — the minimal set of changes that makes `plz` compile, link, parse and
  build under Windows. 17 files, 287 lines. Deliberately incorrect in places:
  `lock_windows.go` returns `nil` (a no-op lock, which would corrupt concurrent builds) and
  `kill_windows.go` kills only the direct child rather than the process tree.

Its value is that it proves the layering described in `../appendix-baseline-errors.md` and
gives M1 a starting shape.

Apply with `git apply docs/design/windows/probe/m1-skeleton.patch` from the repo root, then
build with `-tags forceposix` (see R1 in the appendix).

## Workstream B (`please-build/cc-rules`)

- `cc-rules-windows.patch` — the M5 changes to `build_defs/cc.build_defs`, against v0.7.3.
  Unlike the M1 skeleton these are real and were verified end to end, but they live in
  another repo, so they are recorded here until they are upstreamed.
- `cc-rules.plzconfig_windows_amd64` — the arch config used to test them. The toolchain paths
  assume `g++-mingw-w64-x86-64` is installed.

Reproduce with:

```bash
git clone --branch v0.7.3 https://github.com/please-build/cc-rules
cd cc-rules
git apply /path/to/cc-rules-windows.patch
cp /path/to/cc-rules.plzconfig_windows_amd64 .plzconfig_windows_amd64
plz build --arch windows_amd64 //test/binary:test_binary
file plz-out/bin/windows_amd64/test/binary/test_binary.exe   # PE32+ executable
```
