# Please [![Build Status](https://circleci.com/gh/thought-machine/please.svg?style=shield)](https://circleci.com/gh/thought-machine/please) [![Build Status](https://api.cirrus-ci.com/github/thought-machine/please.svg)](https://cirrus-ci.com/github/thought-machine/please) [![Go Report Card](https://goreportcard.com/badge/github.com/thought-machine/please)](https://goreportcard.com/report/github.com/thought-machine/please) [![Gitter chat](https://badges.gitter.im/thought-machine/please.png)](https://gitter.im/please-build/Lobby)

Please is a cross-language build system with an emphasis on
high performance, extensibility and reproducibility.
It supports a number of popular languages and can automate
nearly any aspect of your build process.

See [please.build](https://please.build) for more information.

Currently Linux (tested on Ubuntu), macOS and FreeBSD are actively supported.

If you're a fan of Please, don't forget to add yourself
to the [adopters](https://github.com/thought-machine/please/blob/master/ADOPTERS.md)
file.

Getting Started
===============

The easiest way to install it on your own machine is to run:
```bash
curl -s https://get.please.build | bash
```
Or, if you prefer, grab one of the tarballs off our
[releases page](https://github.com/thought-machine/please/releases)
and extract it yourself; it typically lives in `~/.please`.

You can also install using [Homebrew](https://brew.sh):
```bash
brew tap thought-machine/please
brew install please
```

Then you simply run `plz init` at the root of your project to set up
a default config and you're ready to go. The best way to get to grips
with Please is through the [codelabs](https://please.build/codelabs.html)!
There's also the [getting started](https://please.build/quickstart.html)
guide that explains the core concepts.

How is it so fast?
==================

Please has a robust and correct caching model that enables us to aggressively
cache artifacts. Caching is based on the hashes of inputs (both files, and
environment variables) to each rule rather than last modified timestamps.
Builds are hermetic so don't have access to anything they haven't explicitly
defined as inputs. This means that if anything changes, we know exactly what
might've been affected, so the minimal set of targets get built and tested.

Because each task is hermetic, they can be run in parallel without any chance
of interfering with each-other. Combine these two concepts with shared remote
caches, and it makes for a blazing fast build system for any language or
technology.

Please is also written in Go and every effort has been made to make it as fast
as possible. There's no startup time waiting to bring up VMs, interpreting code
or communicating with remote JVM processes. The code itself takes full
advantage of Go's concurrency and asynchronicity. The end result is a snappy
command line tool that gets to work immediately and feels great to use.

Why Please, and not Maven, pip, or go build?
============================================

A build system is more than just a mechanism for invoking the compiler.
If you're working on just one language, don't have any code generation,
and don't need to publish any artifacts anywhere, you might not need
Please. Chances are this is not the case.

Building software often involves more than just compiling code. There's
deployment config to template, code to generate, and quite often, there's
more than one language involved. Please provides a powerful, comprehensive,
and understandable framework that you can use to craft a truly holistic
build process.

Please does this through a consistent and seamless command line interface;
there's no need to learn new build systems and technologies for different
languages. Build any target with `plz build`, test any target with `plz test`,
no matter what's going on under the hood.

The [Docker & Kubernetes](https://please.build/codelabs/k8s) codelab covers
building a Kubernetes based application with Please, including reliably
deploying code to a local cluster, and pushing it to a remote registry.

The [genrule()](https://please.build/codelabs/genrule) codelab covers
extending Please with custom build definitions to truly automate any
part of your deployment process.

Why Please, and not make?
=========================

Make is a great tool for running tasks. It's easy enough to understand
because it leaves you very close to the shell. The problem is, it has
limited capability to build out complexity. There have been attempts to
generate make files from higher level tools like cmake and ninja, but they
fall short of what Please sets out to achieve.

The Please build language is a full programming language. There are a high
level set of build rules that make up a declarative DSL to define build
targets, however you can drop into an imperative language that resembles
python when necessary:

```python
subinclude("//my_custom_defs:markdown_page")

pages = []
for page in glob(include = ["*.md"]):
    pages += markdown_page(
        name = page.removesuffix(".md"),
        srcs = [page],
        visibility = ["//website/..."],
    )

go_binary (
    name = "webserver",
    srcs = ["main.go"],
    deps = ["//third_party/go:protobuf"],
    data = pages,
    visibility = ["//services/foo/..."],
)
```

This is distinctively operating at a higher level when compared to make:

```
protobuf:
    go install google.golang.org/protobuf

webserver: protobuf
    go tool compile --pack foo.go -o foo.a

pages: ???
```

Additionally, `make` builds are not hermetic. The above make example installs
protobuf into the host machines Go path. Please builds only have access to files
and environment variables they have explicitly been given access to. You can play
around in the environment targets are built in with `plz build //some/target --shell`.
Additionally, on linux systems, Please can take this a step further with Linux
[namespaces](https://en.wikipedia.org/wiki/Linux_namespaces) to improve
sandboxing especially of the network. Please also has built in task
parallelism so can take full advantage of multi-core machines which were
not a consideration 40 years ago when make was designed.

Finally, Please has a robust caching mechanism base on hashing the
inputs of each rule. Makes cache invalidation is based on the last
modified timestamp which can change unexpectedly forwards and
backwards in time. Combine this with hermetic builds, and Please
caching is never incorrect.


Why Please, not Bazel, Buck or Pants?
=====================================

These build systems are all very similar so choosing between them can be hard.
Please originally replaced buck implementing the subset of features we used. We
found that buck (and the competition) worked great when using the supported
languages but fell flat when breaking new ground.

The biggest difference between Please and the competition is that Please is
designed from the ground up to be extensible. The built-in languages are all
defined in the same
[build language](https://github.com/thought-machine/please/tree/master/rules)
as you use to define your targets, demonstrating that there's nothing special
about them. This puts the build definitions where they should be: in your
domain. You have all the same tools Please has to expand your build
definitions to satisfy your needs.

Please does all this with a focus on simplicity and transparency. There are
a limited amount of concepts that are needed to get started and once they are
grocked, the possibilities are endless. Please relies on these concepts rather
than requiring lots of magic and incantation. Configuration is simple and largely
optional so getting going is easy, and there's no single WORKSPACE file nobody really
owns, with lines of configuration that nobody really understands.

The command line interface is designed with similar considerations in mind. Subcommands
can be added to Please though [aliases](https://please.build/config.html#alias) and
tie into the Please tab-completions. Not only can flags and arguments be
completed, but they can also leverage the build graph to complete labels enabling you
to truly craft your developer experience the way you want it.

Building Please
===============

If you're looking to get involved, check out the contributor
[guidance](CONTRIBUTING.md) to help you get started. If you're a fan of
Please, don't forget to add yourself to the
[adopters](https://github.com/thought-machine/please/blob/master/ADOPTERS.md)
file.

To build Please yourself, run `./bootstrap.sh` in the repo root.
This will bootstrap a minimal version of Please using Go and then
rebuild it using itself.

You'll need to have Go 1.18+ installed to build Please although once
built it can target any version from 1.8+ onwards.

Optional dependencies for various tests include Python, Java, clang,
gold and docker - none of those are required to build components so
their tests will be excluded if they aren't available.

If you'd rather not worry about installing the dependencies, we provide
a prebuilt Docker image based on Ubuntu which is capable of building
the whole thing for you:
[`docker run -it thoughtmachine/please_ubuntu`](https://hub.docker.com/r/thoughtmachine/please_ubuntu)

You can install a locally built copy of Please using `plz install`, or
if it's the first time, by running `./install.sh` after it's built.
This will overwrite whatever you currently have in `~/.please` with
your local version, although you can get back to a released version
again by running `plz update --force`.

To automatically fix linting and code generation issues, run
`plz autofix`.


Documentation
=============

 * [Quickstart](https://please.build/quickstart.html)
 * [Commands & command-line arguments](https://please.build/commands.html)
 * [Built-in rules](https://please.build/lexicon.html)
 * [BUILD language reference](https://please.build/language.html)
 * [Custom build rules](https://please.build/build_rules.html)
 * [Config reference](https://please.build/config.html)
 * [FAQ](https://please.build/faq.html)


Status
======

Please is released & we consider it stable; we follow [semver](https://semver.org)
for releases, so major versions indicate potentially breaking changes to the
BUILD language, command line or other behaviour. We try to minimise this where
possible.

We're very happy to accept pull requests, feature requests, and bugs if it's
not working for you. We don't always have time for everything but Please is
under active development.
