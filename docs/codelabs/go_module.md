summary: Third-party dependencies with go_module() 
description: Set up gRPC and learn how to manage third party dependencies with Please
id: go_module
categories: beginner
tags: medium
status: Published
authors: Jon Poole
Feedback Link: https://github.com/thought-machine/please

# Third-party dependencies with `go_module()`
## Overview
Duration: 1

### Prerequisites
- You must have Please installed: [Install Please](https://please.build/quickstart.html)

### What You'll Learn
In this codelab, we'll be setting up Please to compile third party go modules. You'll learn how to: 
- Use go_module() to download and compile third party go modules
- Download and compile the library and binary parts of a module separately 
- Resolving cyclical dependencies between modules

### What if I get stuck?

The final result of running through this codelab can be found
[here](https://github.com/thought-machine/please-codelabs/tree/main/go_modules) for reference. If you really get stuck
you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## go_module() vs. go_get()
Duration: 3

Please has two ways to download third party dependencies for go. If you're coming to Please for the first time, you 
should be using `go_module()` and can skip this section. If you're currently using `go_get()`, read on. 

The `go_get()` rules existed before go introduced modules, and are temperamental with go 1.15, and simply don't work in 
1.16+. This is due to the new fingerprinting in the go linker. If you're using `go_get()` already, then you will need to 
migrate to `go_module()` as `go_get()` will be deprecated in Please v17. 

### Basic migration
On the surface, `go_get()` and `go_module()` are very similar. The major difference is that `get` on `go_get()` can
be the full import path, including wildcards. The `module` parameter on `go_module()` must be the actual module i.e.
it must match the module in the go.mod. The package to install must be provided in the install list:
```python
go_get(
    name = "some_module_api",
    get = "github.com/someone/some_module/api/...", 
    revision = "v2.0.0",
    module_major_version = "v2", # go_module() can figure this out based on the module to install. 
)
```
Becomes
```python
go_module(
    name = "some_module_api",
    # The module name includes the major version (as it would in their go.mod)
    module = "github.com/someone/some_module/v2",  
    # Version can be a semantic version like this, or it can be a git reference (hash, tag, etc.) 
    version = "v2.0.0",
    # The packages to install must be put here rather than tacked onto the end of the module name
    install = ["api/..."], 
)
```

This approach is more robust and works with any module proxy, not just the ones Please is aware of. For more information 
on `go_module()` and it's sister rule `go_mod_download()`, including how to resolve cyclic dependencies, keep reading 
this guide. 

## Dependencies in Please vs. go build
Duration: 3

If you're coming from a language specific build system like `go build`, Please can feel a bit alien. Please is language 
agnostic so can't parse you source code to automatically update its BUILD files when you add a new import like 
`go mod edit` would for `go build`. 

Instead, you must strictly define all the dependencies of each module. This allows Please to build go modules in a 
controlled and reproducible way without actually having to understand go itself. However, it does take a little more 
work to set up.

A basic `go_module()` usage might look like: 
```python
go_module(
    name = "protobuf_go",
    # By default, we only install the top level package i.e. golang.org/x/sys. To 
    # compile everything, use this wildcard. 
    install = ["..."],
    module = "google.golang.org/protobuf",
    version = "v1.25.0",
    # We must tell Please that :protobuf_go depends on :cmp so we can link to it.  
    deps = [":cmp"],
)

go_module(
    name = "cmp",
    install = ["cmp/..."],
    module = "github.com/google/go-cmp",
    version = "v0.5.5",
)
```

### A note on install
We talk about installing a package. This nomenclature comes from `go install` which would compile a package and 
install it in the go path. In Please terms, this means compiling and storing the result in `plz-out`. We're not 
installing anything system wide. 

The install list can contain exact packages, or could contain wildcards:

```python
go_module(
    name = "module",
    module = "example.com/some/module",
    version = "v1.0.0",
    install = [
       ".", # Refers to the root package of the module. This is the default if no install list is provided. 
       "...", # Refers to everything in the module
       "foo/...", # installs example.com/some/module/foo and everything under it
       "foo/bar", # installs example.com/some/module/foo/bar only
    ]
)
```

## go_mod_download()
Duration: 5

For most modules, you can get away with compiling them in one pass. Sometimes it can be useful to split this out into
separate rules. There are many reasons to do this, for example: to resolve cyclic dependencies; download from a fork 
of a repo; or to vendor a module. 

Another common case is when modules have a `main` package but can also act as a library. One example of this is 
`github.com/golang/protobuf` which contains the protobuf library, as well as the protoc plugin for go. We might want to 
have a binary rule for the protoc plugin, so we can refer to that in our proto config in our `.plzconfig`. 

To do this, we create a `go_mod_download()` rule that will download our sources for us:

```python
go_mod_download(
    name = "protobuf_download",
    module = "github.com/golang/protobuf",
    version = "v1.4.3",
)
```

We can then create a rule to compile the library like so: 
```python
go_module(
    name = "protobuf",
    # Depend on our download rule instead of providing a version
    download = ":protobuf_download",
    install = ["..."],
    module = "github.com/golang/protobuf",
    # Let's skip compiling this package which as we're compiling this separately.
    strip = ["protoc-gen-go"], 
    deps = [":protobuf_go"],
)
```

And then compile the main package under `github.com/golang/protobuf/protoc-gen-go` like so:
```python
go_module(
    name = "protoc-gen-go",
    # Mark this as binary so Please knows it can be executed 
    binary = True,
    # Depend on our download rule instead of providing a version
    download = ":protobuf_download",
    install = ["protoc-gen-go"],
    module = "github.com/golang/protobuf",
    deps = [":protobuf_go"],
)
```

We can then refer to this in our `.plzconfig`:

```
[proto]
ProtocGoPlugin = //third_party/go:protoc-gen-go
```

## Resolving cyclic dependencies
Duration: 5

While go packages can't be cyclically dependent on each other, go modules can. For the most part, this is considered 
bad practice and is quite rare, however the `google.golang.org/grpc` and `google.golang.org/genproto` modules are one 
such example. 

In order to solve this, we need to figure out what parts of the modules actually depend on each other. We can then 
download that module and compile these two parts separately. We will use `go_mod_download()` to achieve this. 

N.B. To run a gRPC service written in go, you will have to install almost all of `google.golang.org/grpc`. For the sake
of brevity, this example only install the subset that `google.golang.org/genproto` needs. You may want to complete this 
by adding `go_module()` rules for the rest of the modules `google.golang.org/grpc` depends on. 

### Installing gRPC's deps
First we must install the dependencies of `google.golang.org/grpc`:
```python
go_module(
    name = "xsys",
    module = "golang.org/x/sys",
    install = ["..."],
    version = "v0.0.0-20210415045647-66c3f260301c",
)

go_module(
    name = "net",
    install = ["..."],
    module = "golang.org/x/net",
    version = "136a25c244d3019482a795d728110278d6ba09a4",
    deps = [
        ":crypto",
        ":text",
    ],
)

go_module(
    name = "text",
    install = [
        "secure/...",
        "unicode/...",
        "transform",
        "encoding/...",
    ],
    module = "golang.org/x/text",
    version = "v0.3.5",
)

go_module(
    name = "crypto",
    install = [
        "ssh/terminal",
        "cast5",
    ],
    module = "golang.org/x/crypto",
    version = "7b85b097bf7527677d54d3220065e966a0e3b613",
)
```

### Finding out what gRPC needs

Next let's try and compile gRPC. We know it has a dependency on some of genproto, but let's set that aside for now:
```python
go_module(
    name = "grpc",
    module = "google.golang.org/grpc",
    version = "v1.34.0",
    # Installing just a subset of stuff to reduce the complexity of this example. You may want to just install "...",
    # and add the rest of the dependencies. 
    install = [
        ".",
        "codes",
        "status",
    ],
    deps = [
        # ":genproto",
        ":cmp",
        ":protobuf",
        ":xsys",
        ":net",
        ":protobuf_go",
    ],
)
```

If we attempt to compile this, we will get an exception along the lines of:
```
google.golang.org/grpc/internal/status/status.go, line 36, column 2: can't find import: "google.golang.org/genproto/googleapis/rpc/status"
```

So let's add `google.golang.org/genproto/googleapis/rpc/...` as a dependency:
```python
go_mod_download(
    name = "genproto_download",
    module = "google.golang.org/genproto",
    version = "v0.0.0-20210315173758-2651cd453018",
)

go_module(
    name = "genproto_rpc",
    download = ":genproto_download",
    install = [
        "googleapis/rpc/...",
    ],
    module = "google.golang.org/genproto",
    deps = [
        ":protobuf",
    ],
)

go_module(
    name = "genproto_api",
    download = ":genproto_download",
    install = [
        "googleapis/api/...",
    ],
    module = "google.golang.org/genproto",
    deps = [
        ":grpc",
        ":protobuf",
    ],
)
```

And update our `:grpc` rule to add `:genproto_rpc` as a dependency:
```python
go_module(
    name = "grpc",
    module = "google.golang.org/grpc",
    version = "v1.34.0",
    # Installing just a subset of stuff to reduce the complexity of this example. You may want to just install "...",
    # and add the rest of the dependencies. 
    install = [
        ".",
        "codes",
        "status",
    ],
    deps = [
        ":genproto_rpc",
        ":cmp",
        ":protobuf",
        ":xsys",
        ":net",
        ":protobuf_go",
    ],
)
```

And if we compile that with `plz build //third_party/go:grpc //third_party/go:genproto_api` we should see they build 
now.

## Using third party libraries
Third party dependencies can be depended on in the same way as `go_library()` rules:
```python
go_library(
    name = "service",
    srcs = ["service.go"],
    deps = ["//third_party/go:net"],
)
```

For more information on writing go code with Please, check out the [go](/codelabs/go_intro) codelab.

## What's next?
Duration: 1

Hopefully you now have an idea as to how to build Go modules with Please. Please is capable of so much more though!

- [Please basics](/basics.html) - A more general introduction to Please. It covers a lot of what we have in this
tutorial in more detail.
- [Built-in rules](/lexicon.html#go) - See the rest of the Go rules as well as rules for other languages and tools.
- [Config](/config.html#go) - See the available config options for Please, especially those relating to the Go language
- [Command line interface](/commands.html) - Please has a powerful command line interface. Interrogate the build graph,
determine files changes since master, watch rules and build them automatically as things change and much more! Use
`plz help`, and explore this rich set of commands!

Otherwise, why not try one of the other codelabs!