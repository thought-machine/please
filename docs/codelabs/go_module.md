summary: Third-party dependencies with Puku
description: Add, update, pin, and remove Go third-party dependencies using go get and plz puku (no go_module())
id: go_module
categories: beginner
tags: medium
status: Published
authors: Jon Poole
Feedback Link: https://github.com/thought-machine/please

# Third-party dependencies with Puku

## Overview
Duration: 2

Notes: `go_module()` is deprecated in Core3. This codelab teaches a practical workflow that uses standard Go tooling (`go get` / `go mod`) together with Puku to generate and maintain third-party go targets (`go_repo`).

### Goals
- Add a new third‑party dependency with `go get`
- Sync the dependency into Please with `plz puku sync`
- Let puku update BUILD deps with `plz puku fmt`
- Upgrade, pin/exclude, and remove modules safely
- Diagnose missing import / missing subrepo issues

You will not use `go_module()` in this guide.

### Prerequisites
- Please installed and configured: https://please.build/quickstart.html
- Go 1.20+ installed and on PATH
- Puku available in one of the following ways:
  - Via Please alias: add an alias to `.plzconfig` (see below), or
  - Installed locally (if the first doesn't work, try the second):
    - `go install github.com/please-build/puku/cmd/puku@latest`
    - `go get github.com/please-build/puku/cmd/puku`

### What you’ll learn
- Add and upgrade dependencies with `go get`
- Sync `go.mod` into `third_party/go/BUILD` with `plz puku sync`
- Let `plz puku fmt` add third-party deps to your BUILD targets
- Diagnose missing imports and missing subrepos
- Pin or exclude dependency versions with `go mod edit`
- Remove third-party modules safely

### What if I get stuck?

The final result of running through this codelab can be found
[here](https://github.com/thought-machine/please-codelabs/tree/main/go_modules) for reference. If you really get stuck
you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## Initialising your project and running puku with please
Duration: 5

The easiest way to get started is from an existing Go module:

```bash
mkdir puku_sync && cd puku_sync
go mod init example_module
plz init --no_prompt
plz init plugin go
```

Define a valid Puku version number as a build configuration string in `.plzconfig`:

```
[BuildConfig]
puku-version = "1.17.0"
```

Uncomment and edit the following lines in your `.plzconfig` to set up `please` version:

```
[please]
version = 17.22.0
```

Configure a Please alias for Puku (optional but convenient):

```
[Alias "puku"]
Cmd = run //third_party/binary:puku --
PositionalLabels = true
Desc = A tool to update BUILD files in Go packages
```

With the alias, you can use `plz puku` instead of `plz run //third_party/binary:puku`.

Then download that version of Puku in `third_party/binary/BUILD`:

```python
remote_file(
    name = "puku",
    url = f"https://github.com/please-build/puku/releases/download/v{CONFIG.PUKU_VERSION}/puku-{CONFIG.PUKU_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
    binary = True,
)
```

Configure the Go plugin to point at your go.mod (recommended). Create a repo-root `BUILD` with a filegroup for go.mod:

1) Add a filegroup for go.mod at `BUILD` in repo root:
```python
filegroup(
    name = "gomod",
    srcs = ["go.mod"],
    visibility = ["PUBLIC"],
)
```

2) Update your `.plzconfig`:
```
[Plugin "go"]
Target = //plugins:go
ModFile = //:gomod
```

This lets Puku use standard `go get` to resolve modules, then sync them into `third_party/go/BUILD`.

### Configuring the PATH for Go

By default, Please looks for Go in the following locations:
```
/usr/local/bin:/usr/bin:/bin
```

If you installed Go elsewhere (e.g., via Homebrew on macOS, or a custom location), you must configure the path in `.plzconfig`.

First, find where your Go binary is located:
```bash
which go
```

Then add the path to `.plzconfig`. For example, if Go is at `/opt/homebrew/bin/go`:

```ini
[Build]
Path = /opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin
```

Or if it's at `/usr/local/go/bin/go`:

```ini
[Build]
Path = /usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
```

**Note:** On Windows, use `where.exe go` to find the Go installation path.

### Installing the Go standard library (Go 1.20+)

From Go version 1.20 onwards, the standard library is no longer included by default with the Go distribution. You must install it manually:

```bash
GODEBUG="installgoroot=all" go install std
```

## Adding and updating modules
Duration: 5

Let's add a new third-party dependency using `go get` and sync it with Puku.

### Adding a new module

First, let's create a simple Go program that uses a third-party library. Create a file `src/hello/hello.go`:

```go
package main

import (
	"fmt"
	"github.com/google/uuid"
)

func main() {
	id := uuid.New()
	fmt.Printf("Generated UUID: %s\n", id.String())
}
```

Now add the dependency with `go get`:

```bash
GOTOOLCHAIN=local go get github.com/google/uuid
```

Sync the changes to `third_party/go/BUILD`:

```bash
plz puku sync -w
```

This creates a `go_repo()` rule in `third_party/go/BUILD` for the `uuid` module. You may need to create the `third_party/go/BUILD` file if it doesn't exist.

### Creating the BUILD file

Create `src/hello/BUILD`:

```python
go_binary(
    name = "hello",
    srcs = ["hello.go"],
)
```

Now let Puku automatically add the dependency:

```bash
plz puku fmt //src/hello
```

Puku will update your BUILD file to include the dependency on `//third_party/go:google-uuid` (or the subrepo format).

Build and run your program:

```bash
plz run //src/hello
```

### Updating an existing module

To update a module to a specific version:

```bash
GOTOOLCHAIN=local go get github.com/google/uuid@v1.6.0
plz puku sync -w
```

To update to the latest version:

```bash
GOTOOLCHAIN=local go get -u github.com/google/uuid
plz puku sync -w
```

After syncing, rebuild your targets to use the updated version.

### Troubleshooting

**Missing import error?** If you see `could not import ... (open : no such file or directory)`, the module providing that package is missing. Add it with:

```bash
go get <module-name>
plz puku sync -w
```

**Missing subrepo error?** If you see `Subrepo ... is not defined`, you need to add or migrate the module:

```bash
go get <module-name>
plz puku sync -w
```

## Stop a module from updating
Duration: 3

Sometimes you need to prevent a module from being updated due to breaking changes or compatibility issues.

### Excluding a specific version

Use the `exclude` directive to prevent a specific version from being used:

```bash
go mod edit -exclude github.com/example/module@v2.0.0
plz puku sync -w
```

This prevents version `v2.0.0` from being selected. Go will use the next highest non-excluded version.

To remove an exclusion:

```bash
go mod edit -dropexclude github.com/example/module@v2.0.0
plz puku sync -w
```

### Pinning to a specific version

Use the `replace` directive to pin a module to a specific version:

```bash
go mod edit -replace github.com/example/module=github.com/example/module@v1.5.0
plz puku sync -w
```

This pins the module to `v1.5.0` regardless of what other dependencies require.

To unpin (and upgrade at the same time):

```bash
go mod edit -dropreplace github.com/example/module
go get -u github.com/example/module
plz puku sync -w
```

**Warning:** Pinning modules can cause compatibility issues with other dependencies. Use sparingly and resolve as soon as possible.

### Example scenario

Let's say a new version of `uuid` has a breaking change. Pin it to a working version:

```bash
go mod edit -replace github.com/google/uuid=github.com/google/uuid@v1.3.0
plz puku sync -w
plz build //src/hello
```

## Removing modules
Duration: 3

Before removing a module, ensure it's not used anywhere in your codebase.

### Steps to remove a module

1. **Verify no dependencies exist:**

```bash
plz query revdeps //third_party/go:module_name --level=-1 | grep -v //third_party/go
```

If this returns no results, the module is safe to remove.

2. **Remove the `go_repo()` target from `third_party/go/BUILD`:**

Open `third_party/go/BUILD` and delete the corresponding `go_repo()` rule.

3. **Remove from `go.mod` and `go.sum`:**

```bash
go mod edit -droprequire github.com/example/module
go mod tidy
```

4. **Sync the changes:**

```bash
plz puku sync -w
```

**Note:** Puku does not currently automate module removal, so this process is manual.

### Example

Let's say we want to remove an unused module:

```bash
# Check for dependencies
plz query revdeps //third_party/go:unused_module --level=-1 | grep -v //third_party/go

# If safe, remove from go.mod
go mod edit -droprequire github.com/unused/module
go mod tidy

# Manually delete the go_repo() rule from third_party/go/BUILD
# Then sync
plz puku sync -w
```

## Using new modules
Duration: 4

Once you've added a module with `go get` and `plz puku sync`, you can use it in your code.

### Automatic dependency management

The easiest way is to let Puku handle dependencies automatically:

1. Import the package in your `.go` file
2. Run `plz puku fmt //your/package`

Puku will parse your imports and add the necessary dependencies to your BUILD file.

### Manual dependency specification

There are two ways to specify dependencies on third-party packages:

**1. Subrepo convention (recommended):**

```python
go_library(
    name = "mylib",
    srcs = ["mylib.go"],
    deps = [
        "///third_party/go/github.com_google_uuid//",
    ],
)
```

The subrepo format is: `///third_party/go/<module_path_with_underscores>//<package_path>`

**2. Install list (go_module style):**

Add packages to the `install` list on the `go_repo()` target:

```python
go_repo(
    name = "google-uuid",
    module = "github.com/google/uuid",
    version = "v1.6.0",
    install = ["."],  # Installs the root package
)
```

Then depend on it like:

```python
go_library(
    name = "mylib",
    srcs = ["mylib.go"],
    deps = ["//third_party/go:google-uuid"],
)
```

### Watch mode

For active development, use watch mode to automatically update BUILD files as you code:

```bash
plz puku watch //src/...
```

This watches for changes to `.go` files and updates dependencies automatically.

### Best practices

- Use `plz puku fmt` to keep dependencies up to date
- Use the subrepo format for better build incrementality
- Review changes before committing to avoid unexpected version changes
- Run `plz test` after adding/updating dependencies to catch issues early

## What's next?
Duration: 1

Congratulations! You now know how to manage Go third-party dependencies using `go get` and Puku.

### Learn more

- [Puku GitHub repository](https://github.com/please-build/puku) - Complete Puku reference
- [Please basics](/basics.html) - A more general introduction to Please. It covers a lot of what we have in this tutorial in more detail.
- [Go plugin rules](/plugins.html#go) - See the rest of the Go plugin rules and config.
- [Built-in rules](/lexicon.html#go) - See the rest of the built in rules.
- [Config](/config.html) - See the available config options for Please.
- [Command line interface](/commands.html) - Please has a powerful command line interface. Interrogate the build graph, determine file changes since master, watch rules and build them automatically as things change, and much more! Use `plz help`, and explore this rich set of commands!

Otherwise, why not try one of the other codelabs!
