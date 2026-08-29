# AGENTS.md

This file provides guidance to coding agents when working with code in this repository.

## What this is

Please is a cross-language build system written in Go. This repo builds itself: `plz` is used to
build `plz`. That bootstrap relationship shapes nearly everything below — most workflows involve
building a new `please` binary and then running it against this same repo.

## Commands

Building requires an existing Please. `./pleasew` downloads/uses one if you don't have `plz` on
your `$PATH`; substitute it for `plz` in any command below.

```bash
./bootstrap.sh                 # Build please from scratch with `go run`, then run all tests
./bootstrap.sh --skip_tests    # Just the build (alias: plz bootonly)
plz build //src:please         # Incremental rebuild -> plz-out/bin/src/please
plz test //src/...             # Run tests
./test.sh                      # What bootstrap.sh runs: unit tests, then e2e separately
```

Test selection:

```bash
plz test //src/core:core_test              # One target
plz test //src/core:core_test TestFoo      # One test within a target (positional arg = selector)
plz test //src/... --exclude=e2e           # Skip end-to-end tests
plz test //test/... --include=e2e          # Only end-to-end tests
plz test //src/core:core_test --rerun      # Force rerun even if the hash is unchanged
plz cover //src/...                        # Tests with coverage
```

Results land in `plz-out/log/test_results.xml`; logs in `plz-out/log/`.

Lint and format:

```bash
plz lint          # golangci-lint over src/... and tools/..., plus a `plz fmt` check
plz autofix       # plz fmt -w, gofmt -s -w src tools test, and regenerate codegen
plz fmt -w        # Format BUILD files only
plz puku sync     # Reconcile third_party/go BUILD files with go.mod
```

Testing a change to Please against the repo itself:

```bash
plz plz build //src/...   # `plz plz` runs the freshly-built in-repo please (alias in .plzconfig)
plz install               # Install the built version into ~/.please
```

All tests should be run via plz. Do not use `go test` or `go build` directly unless you have
a specific reason to; there are differences between how plz and go act which means that some
tests may not work as expected if run directly with `go`.

## Architecture

`src/please.go` is the CLI entry point — a big `go-flags` struct defining every subcommand. Each
subcommand assembles a `core.BuildState` and hands it to `plz.Run` (`src/plz/plz.go`), which is the
orchestrator: it spins up worker pools bounded by local and remote limiters and pulls from the task
queues that `BuildState` exposes.

The three pipeline stages each have their own package and their own README worth reading before
changing them:

- **`src/core`** — `BuildTarget`, `BuildLabel`, `BuildGraph`, `Configuration`, `Subrepo`, and
  `BuildState`. Everything hangs off this. The target queuing/activation logic here is the most
  subtle code in the repo: targets are added to the graph inactive, activated when needed, and
  progress through an ordered `BuildTargetState` enum that also acts as the synchronisation
  primitive. All queuing funnels through `QueueTarget()`.
- **`src/parse`** — parses BUILD files into graph targets. `src/parse/asp` is a hand-written
  interpreter for the BUILD language: a Python subset (no `import`/`try`/`class`/`while`, no floats
  or sets, string-keyed dicts only, immutable-ish lists). Its README lists the exact divergences.
  Parsing synchronises on `state.SyncParsePackage(label)` — a nil return means you own the parse.
- **`src/build`** — builds a target. Three-way check: already built in `plz-out`, restorable from
  cache, or needs a real build. Complicated by post-build functions and output dirs, which can
  change the output hash after the fact, so metadata files are fetched and replayed before the
  outputs are.
- **`src/test`** — runs tests and collates results into `core.TestSuite`s in a Surefire-like XML
  format. Handles flaky reruns and `--num_runs` by keeping multiple `TestExecution`s per `TestCase`.

Supporting packages: `src/remote` implements the Bazel remote execution API client; `src/cache` has the
dir/http/cmd cache backends; `src/output` and `src/cli` handle the interactive display; `src/fs`,
`src/process`, `src/sandbox` are the OS-level layers. `rules/*.build_defs` are the built-in build
rules, written in the BUILD language itself and embedded into the binary.

Caching is hash-based, not timestamp-based: hashes cover rule definition, config, sources, and
secrets. Anything that changes what a target produces must be folded into its hash, or you will
create stale-cache bugs that only show up on other people's machines.

## Conventions

- Language plugins (go, cc, python, shell) are external repos pinned in `plugins/BUILD` and
  preloaded via `.plzconfig`. Bumping a plugin version is a `plugins/BUILD` edit.
- Third-party Go deps are BUILD targets under `third_party/go`, managed by `puku` from `go.mod`.
  Don't hand-edit them; run `plz puku sync`.
- The end-to-end tests in `test/` invoke `plz` recursively against this repo. They are deliberately
  run in a separate pass (see `test.sh`) with the lock disabled and sandboxing off, and they assert
  on exact output text, so they're brittle — expect to update `.txt` golden files. `plz_e2e_test`
  in `test/build_defs/test.build_defs` is the macro they all use.
- Some tests need toolchains that may be absent (python3, xz); `test.sh` detects this and passes
  `--exclude` flags. If a test fails locally but not in CI, check whether it's one of these.
- Releases: bump `VERSION` and add a `ChangeLog` entry in the existing format (version heading,
  then bullets with PR numbers).
- `tree/` is a generated perf-test repo, blacklisted from parsing; `plz-out/` is all build output.
