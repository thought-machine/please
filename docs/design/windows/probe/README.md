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
