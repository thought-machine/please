summary: Tips and tricks - plz query
description: Tips and tricks to help you become productive with Please - using plz query to query the build graph
id: plz_query
categories: intermediate
tags: medium
status: Published
authors: Jon Poole
Feedback Link: https://github.com/thought-machine/please

# Tips and tricks - plz query

## Overview

Duration: 2

### Prerequisites

- You must have Please installed: [Install please](https://please.build/quickstart.html)
- You should have a basic understanding of using Please to build and test code

### What you'll learn

This codelab isn't exhaustive however it should give you an idea of the sort of
things the Please CLI is capable of:

- Finding the dependencies of a target
- Including and excluding targets
- Printing information about targets as well as internal targets

## Setting up

Duration: 2

For this codelab we will be using the Please codelabs repo:

```bash
$ git clone https://github.com/thought-machine/please-codelabs
Cloning into 'please-examples'...
remote: Enumerating objects: 228, done.
remote: Total 228 (delta 0), reused 0 (delta 0), pack-reused 228
Receiving objects: 100% (228/228), 38.23 KiB | 543.00 KiB/s, done.
Resolving deltas: 100% (79/79), done.
```

We'll be using the getting started with go codelab for these examples:

```bash
cd please-codelabs/getting_started_go
```

## Finding dependencies of a target

Duration: 4

Please has a strict build graph representing each build target and their
dependencies on each other. Among many things, this graph can be interrogated
to determine the dependencies of a target:

```bash
$ plz query deps //src/greetings:greetings_test 
//src/greetings:greetings_test
  //src/greetings:greetings
    ///go//tools:please_go
      //plugins:go
        //_please:arcat
    //third_party/go:toolchain
  //third_party/go:testify
    ///third_party/go/github.com_stretchr_testify//:installs
      ///third_party/go/github.com_stretchr_testify//assert:assert
        ///third_party/go/github.com_davecgh_go-spew//spew:spew
          //third_party/go:github.com_davecgh_go-spew
        ///third_party/go/github.com_pmezard_go-difflib//difflib:difflib
          //third_party/go:github.com_pmezard_go-difflib
        ///third_party/go/gopkg.in_yaml.v3//:yaml.v3
          //third_party/go:gopkg.in_yaml.v3
      ///third_party/go/github.com_stretchr_testify//require:require
```

This can be especially useful when trying to improve build performance.
Unnecessary dependencies between targets can cause certain rules to be rebuilt
when they don't need to be.

### Subrepo rules

Woah, what are these `///third_party/go/foo//:bar` targets? Targets that begin
with a `///` are subrepo targets. In this case, the third-party dependency
*testify* has been defined using a `go_repo()` rule, which downloads the go
module into plz-out, generates Please BUILD files for each of its packages, and
then builds it like any other Please project. So
`///third_party/go/github.com_stretchr_testify//assert:assert` is saying "look
in the subrepo called *third_party/go/github.com_stretchr_testify*, and retrieve
for me the build target `//assert:assert`.

You can `plz query print` these targets just like you would any other target:

```bash
plz query print ///third_party/go/github.com_stretchr_testify//assert:assert
```

This will show you the underlying build rule for that target. Or, if you prefer,
you could have a look in the plz-out directory at the generated build rule:

```bash
$ cat plz-out/subrepos/third_party/go/github.com_stretchr_testify/assert/BUILD
subinclude("///go//build_defs:go")

go_library(
    name = "assert",
    srcs = [
        "assertion_compare.go",
        "assertion_compare_can_convert.go",
        "assertion_format.go",
        "assertion_forward.go",
        "assertion_order.go",
        "assertions.go",
        "doc.go",
        "errors.go",
        "forward_assertions.go",
        "http_assertions.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_davecgh_go-spew//spew",
        "///third_party/go/github.com_pmezard_go-difflib//difflib",
        "///third_party/go/gopkg.in_yaml.v3//:yaml.v3",
    ],
)
```

### Internal rules

If you pass the `--hidden` flag to a `plz query` command, you'll come across
*internal* targets as well. These can be identified by the leading `_` in their
name. E.g.

```bash
$ plz query deps //src/greetings:greetings --hidden
//src/greetings:greetings
  //src/greetings:_greetings#import_config
  //src/greetings:_greetings#pkg_info
    ///go//tools:please_go
      //plugins:go
        //_please:arcat
        //plugins:_go#download
  //src/greetings:_greetings#srcs
  //third_party/go:toolchain
    //third_party/go:_toolchain#download
```

As always, we can inspect these with `plz query print`, e.g.

```bash
$ plz query print //src/greetings:_greetings#srcs
# //src/greetings:_greetings#srcs:
filegroup(
    name = '_greetings#srcs',
    srcs = ['greetings.go'],
    labels = [
        'link:plz-out/go/src/${PKG}',
        'go_src',
        'go',
    ],
    visibility = ['//src/...'],
    build_timeout = 600,
    requires = ['go'],
)
```

This particular internal rule is a filegroup that was generated by
`go_library()` and is used to expose the Go source files that make up that
library. You shouldn't depend on these types of rules directly as they may
change between minor releases of Please.

## Reverse dependencies

Duration: 2

If you're changing a build rule that you know has a wide-reaching effect, it
might be good to run all the tests that will be affected by that change. Let's
find the reverse dependencies of our subrepo rules:

```bash
$ plz query revdeps ///third_party/go/github.com_stretchr_testify//require:require
///third_party/go/github.com_stretchr_testify//:installs
```

Well that doesn't look quite right... We should see
`//src/greetings:greetings_test` too.

Turns out finding reverse dependencies is quite a slow operation. Please limits
this to just one level so you don't accidentally lock up your terminal trying to
walk the whole build graph. You can set the level with `--level=2` or if you
want to get all reverse dependencies, you can set it to `-1`:

```bash
$ plz query revdeps ///third_party/go/github.com_stretchr_testify//require:require --level -1
//src/greetings:greetings_test
//third_party/go:testify
///third_party/go/github.com_stretchr_testify//:installs
```

Be careful, this can be slow on larger build graphs. You can use
`--include=//src/foo/...` to limit the search to a slice of your repository.
More on this later in this codelab!

## Composing plz commands

Duration: 2

So we've managed to determine which targets will be affected by our change. How
do we run these tests? Please can be instructed to listen for targets on
standard input:

```bash
$ plz query revdeps ///third_party/go/github.com_stretchr_testify//require:require --level -1 | plz test -
//src/greetings:greetings_test 1 test run in 8ms; 1 passed
1 test target and 1 test run; 1 passed.
Total time: 6.62s real, 10ms compute.
```

The `-` at the end of `plz test -` indicates to Please that we will be
supplying the targets to build over standard input.

## Including and excluding targets

Duration: 2

Almost all Please commands can take in the `--include` and `--exclude`
arguments. These can be used to specifically exclude targets:

```bash
$ plz query revdeps --exclude //src/greetings:greetings_test --level=-1 ///third_party/go/github.com_stretchr_testify//require:require | plz test -
0 test targets and 0 tests run; 0 passed.
Total time: 40ms real, 0s compute.
```

As you can see, we excluded the test from earlier so `plz test` didn't run it.
We can also exclude this on the test command:

```bash
$ plz query revdeps --level=-1 ///third_party/go/github.com_stretchr_testify//require:require | plz test --exclude //src/greetings:greetings_test -
0 test targets and 0 tests run; 0 passed.
Total time: 40ms real, 0s compute.
```

### Including based on label

Targets can be labeled in Please. Most of the built-in rules apply some basic
labels, e.g. the Go rules apply the `go` label to their targets. These can be
very useful to run all tests for a given language:

```bash
plz build --include go --exclude //third_party/go/...
```

This will build all Go targets but will only build targets under
`//third_party/go/...` if they're a dependency of a target that needs to be built.

You may also add custom labels to your targets. Update `srcs/greetings/BUILD` as such:

### `src/greetings/BUILD`
```python
go_library(
    name = "greetings",
    srcs = ["greetings.go"],
    visibility = ["//src/..."],
    labels = ["my_label"], # Add a label to the library rule
)

go_test(
    name = "greetings_test",
    srcs = ["greetings_test.go"],
    deps = [
        ":greetings",
        "//third_party/go:assert",
    ],
    external = True,
)
```

```bash
$ plz query alltargets --include=my_label
//src/greetings:greetings

$ plz build --include=my_label
Build finished; total time 300ms, incrementality 100.0%. Outputs:
//src/greetings:greetings:
  plz-out/gen/src/greetings/greetings.a
```

This can be especially useful for separating out slow running tests:

```bash
plz test --exclude e2e
```

## What's next?

Duration: 1

Hopefully this has given you a taster for what is possible with `plz query`,
however there's so much more. See the [cli](/commands.html#query) for an idea of
what's possible!
