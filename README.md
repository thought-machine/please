# Please [![Build Status](https://circleci.com/gh/thought-machine/please.svg?style=shield)](https://circleci.com/gh/thought-machine/please) [![Build Status](https://api.cirrus-ci.com/github/thought-machine/please.svg)](https://cirrus-ci.com/github/thought-machine/please) [![Go Report Card](https://goreportcard.com/badge/github.com/thought-machine/please)](https://goreportcard.com/report/github.com/thought-machine/please) [![Gitter chat](https://badges.gitter.im/thought-machine/please.png)](https://gitter.im/please-build/Lobby)

Please is a cross-language build system with an emphasis on
high performance, extensibility and reproducibility.
It supports a number of popular languages and can automate
nearly any aspect of your build process.

See http://please.build for more information.

Currently Linux (tested on Ubuntu), macOS and FreeBSD are actively supported.


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
a default config and you're good to start adding BUILD files.
See [the website](http://please.build) for more instructions about
how to write them.

There are various commands available to interact with Please, the
most obvious & useful ones initially are `plz build` and `plz test`
to build things & run tests respectively. See `plz --help` or the
[documentation](https://please.build/commands.html) for more comprehensive
information.


Building Please
===============

To build Please yourself, run `./bootstrap.sh` in the repo root.
This will bootstrap a minimal version of Please using Go and then
rebuild it using itself.
You'll need to have Go 1.13+ installed to build Please although once
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

We're very happy to accept pull requests & feature requests, and bugs if it's
not working for you. We don't always have time for everything but please is
under active development.
