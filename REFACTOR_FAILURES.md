# Outstanding `//test/...` failures from the `src/plz/plz.go` refactor

Status as of the run below. 12 failing test targets, grouped by root cause.

```
plz plz test //test/... //src/... -o test.timeout:60s --exclude benchmark
261 test targets and 1041 tests run; 1027 passed, 12 errored, 2 skipped.
```

All of `//src/...` passes. The failures are all in `//test/...` (e2e).

## Already fixed

Recorded for context; these are done and are not in the 12 below.

| Problem | Symptom | Fix |
| --- | --- | --- |
| `buildOne` re-derived its second errgroup from a context that `Wait()` had already cancelled | Large builds stopped early with `context canceled` and no error | Don't shadow `ctx`; use a separate `gctx` per group |
| `buildAll` called `buildOne` directly, bypassing the `buildOnce` memo | `panic: close of closed channel` in `FinishBuild` when a target was reached both via `:all` and as a dependency | Route through `Build` |
| `Build` claimed the `buildOnce` entry around `parseTarget` | Deadlock when parsing a package re-entered `Build` for a label defined in that same file (`subinclude(":defs_2")`) | Claim the entry after parsing, so only the build phase is memoised |
| `RuntimeDependencies()` widened to include data deps | `query print -f runtime_deps` printed data deps too | Split into `RuntimeDependencies()` and `RuntimeAndDataDependencies()` |
| `AddOriginalTarget` called inside the task goroutine | Nondeterministic original-target order, visible as `.gitignore` write order | Call it synchronously in `queueTask`, in walk order |

## A. A failed package parse strands every other waiter
# DONE in 89ff16978efcc727a8d1272eedecd00c2c8288d6 and d2a0d2dd773a8d651d9fac991a04a2a7e67a292b

**Failing:** `//test/preloaded_subinc:preload_subinc_test` (hangs until the test timeout)

`BuildGraph.PackageOrWait` goes through `cmap.Map.GetOrWait`: the first caller claims the
slot and everyone else gets a wait channel. If that first caller's parse *fails*, the package
is never `Set`, so the channel is never closed and the other waiters block forever.

In this test, preload `//plugins:shell` fails with `build file not found: ... there's no BUILD
file in plugins/`, and `///pleasings//k8s:k8s` then waits on package `plugins` indefinitely:

```
core.(*BuildGraph).PackageOrWait   src/core/graph.go:117
plz.(*runner).Parse                src/plz/plz.go:144
plz.(*runner).Parse                src/plz/plz.go:158
plz.(*runner).parseTarget          src/plz/plz.go:211
plz.(*runner).Build                src/plz/plz.go:337
plz.(*runner).RegisterPreloads     src/plz/plz.go:525
```

The packages map has no error path, unlike `cmap.ErrMap` which `buildOnce` uses — a failing
parse has no way to publish its failure to waiters. Giving `graph.packages` the same
error-carrying treatment looks like the fix.

`RegisterPreloads` now takes a context (it previously used a bare `errgroup.Group`), so a
failing preload can at least cancel its siblings. That turns the hang into an error but
doesn't fix the underlying stranding — any parse failure anywhere still relies entirely on
context cancellation to release waiters.

## B. `runner.Parse` lost the `dependent` label
DONE in 0ba3093a522688f55a0025c02ff43a0f622b3809
**Failing:** `//test/subrepo/same_package_error:same_package_error` (hangs until the test timeout)

The test asserts the error `subrepo X is not defined in this package yet` for a BUILD file
that subincludes from a subrepo it declares further down:

```python
subinclude("///subrepo//:foo")

local_repository(name = "subrepo", path = "subrepo")
```

The old `parse.checkSubrepo` guarded this with `inSamePackage(sl, dependent)`. The current
check at `src/plz/plz.go:154` compares `sl` against `label` instead:

```go
if sl.Subrepo == label.Subrepo && sl.PackageName == label.PackageName {
    // TODO(peter): Unsure if this is a legit case or not.
    return nil, fmt.Errorf("subinclude from within same package of a subrepo")
}
```

For `///subrepo//:foo`, `sl` is `//:subrepo`, so `sl.Subrepo` (`""`) never equals
`label.Subrepo` (`"subrepo"`) and the guard can't fire. We then recurse into `r.Parse(sl)`
and land in `PackageOrWait` for the package this goroutine is *already* parsing.

The comparison needs the dependent, not the label — which means reversing the
`TODO(peter): can we drop the dependent here?` at `src/plz/plz.go:163`
(`parse.Parse(r.state, label, label)`). The answer appears to be no.

## C. `CheckArchSubrepo` is consulted first, and it accepts anything
SORT OF DONE in 0ba3093a522688f55a0025c02ff43a0f622b3809
DONE in ba3fef2921b215d29ced22bb5336e65fd70fbbbc
**Failing:** `//test/proto_plugin:proto_rules_test`

```
CRITICAL: Found multiple definitions for subrepo 'go_proto'
  (... Target://plugins:go-proto ... Arch:{OS:linux Arch:amd64} IsCrossCompile:false)
  (... Target:<nil>              ... Arch:{OS:go    Arch:proto} IsCrossCompile:true)
```

`cli.Arch.UnmarshalFlag` accepts any `x_y` string with no `/` or `@`, so `go_proto` parses as
OS=`go`, Arch=`proto`. `src/plz/plz.go:152` checks that *before* trying to resolve the real
subrepo definition, registering a phantom arch subrepo that later collides with the genuine
one from `//plugins:go-proto`.

Previously `CheckArchSubrepo` was only a fallback, reached from `maybeParseSubrepoPackage`
after the normal subrepo lookup had already failed. Restoring that ordering is the fix, but
it needs care: the failed normal lookup goes through `parse.Parse`, which calls
`state.LogBuildError`, so a naive "try normal, fall back to arch" would log a spurious parse
error for every legitimate arch subrepo.

## D. Cross-compiling doesn't take effect

**Failing:** `//test/cross_compile:location_cross_compile_test`, `//test/cross_compile:x86_test`

```
Unexpected contents of file: test/cross_compile/arch.txt
unexpected architecture of binary
```

Targets under `///linux_x86//` are being built for the host architecture. Plausibly downstream
of C, since the arch subrepo is now registered from a different place, but that is unverified —
this one needs its own investigation.

## E. `BuildTarget.Dependencies` panics on lazily-resolved deps

**Failing:** `//test/export:export_src_please_test`, `//test/export/test_go_bin:export_go_target`,
`//test/export/test_native_target_with_go_dep:export_native_target_with_go_dep`

`src/core/build_target.go:598` resolves each declared dep with `graph.TargetOrDie`. Since
dependency resolution moved out of `build_target`, the graph legitimately may not contain a
declared dep, and the lookup panics. Two callers hit it:

```
panic: Target ///third_party/go/github.com_dustin_go-humanize//:go-humanize not found in build graph
core.(*BuildTarget).Dependencies   src/core/build_target.go:598
export.(*export).export            src/export/export.go:189
export.ToDir                       src/export/export.go:51
```

```
panic: Target //_please:arcat not found in build graph
core.(*BuildTarget).Dependencies   src/core/build_target.go:598
core.(*cycleDetector).Check.func1  src/core/cycle_detector.go:38
core.(*BuildState).checkForCycles  src/core/state.go:537
```

Note the second one fires from a background goroutine, so it takes the process down
regardless of what the build is doing.

## F. Post-build-added runtime deps are never built

**Failing:** `//test/runtime_deps:runtime_deps_test`, `//test/plz_run:_add_data_target_test`

```
ERROR: //test:runtime_deps_test_case failed:
  cannot calculate hash for plz-out/gen/test/post_build_runtime_dep: file does not exist
ERROR: //src:_add_data_file_and_target failed:
  cannot calculate hash for plz-out/gen/src/foo.txt: file does not exist
```

`buildOne` snapshots the deps *before* the target itself is built:

```go
deps := slices.Collect(target.RuntimeAndDataDependencies())
```

so anything a post-build function adds via `add_runtime_dep` / `add_data` is missed. This is
the existing `TODO(peter): Need to handle targets getting extra deps added by post-build
functions here.` at `src/plz/plz.go:277` — the collect needs redoing once the target has
finished building.

## G. `--keep_going` is not implemented

**Failing:** `//test/keep_going:plz_build_all`

```
Build stopped after 40ms. 2 targets failed: //package:fail, //package:fail2
plz-out/gen/package/dep_pass doesnt exist
```

`state.KeepGoing` is set in `src/please.go:1149` and read nowhere in `src/plz/plz.go`. The
errgroups short-circuit on the first error, so unrelated targets like `//package:dep_pass`
never get built. An `errgroup` can't express this on its own — the group needs to run
everything and collect errors when `KeepGoing` is set.

## Suggested order

A and B first: both are hangs, and both come down to the parse path waiting on a channel that
nothing guarantees will be closed. C then D, since D may fall out of C.
