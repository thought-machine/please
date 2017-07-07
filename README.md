# Please [![Build Status](https://circleci.com/gh/thought-machine/please.svg?style=shield)](https://circleci.com/gh/thought-machine/please) [![Go Report Card](https://goreportcard.com/badge/github.com/thought-machine/please)](https://goreportcard.com/report/github.com/thought-machine/please)

Please is a cross-language build system with an emphasis on
high performance, extensibility and reproduceability.
It supports a number of popular languages and can automate
nearly any aspect of your build process.

See http://please.build for more information.

Currently Linux (tested on Ubuntu) and OSX are actively supported,
and FreeBSD is known to work with a little setup (see FAQ for details).


Getting Started
===============

The easiest way to install it on your own machine is to run:
```bash
curl -s https://get.please.build | bash
```
In order for it to run you will need a Python interpreter available.
You can use either PyPy or CPython with cffi; on Linux Please will
attempt to download a portable version of PyPy if a system one isn't
available, on OSX installation is fairly straightforward with Homebrew,
simply run `brew install pypy`.

Alternatively, you've likely got CPython installed already, and installing
cffi is as simple as `sudo pip install "cffi>=1.5.0"`.

Then you simply run `plz init` at the root of your project to set up
a default config and you're good to start adding BUILD files.
See [the website](http://please.build) for more instructions about
how to write them.


Building Please
===============

To build Please yourself, run `./bootstrap.sh` in the repo root.
This will set up the minimal environment needed to build Please,
build it once manually and then rebuild it again using itself.
You'll need to have Go 1.8+ installed to build Please (we use some
new standard library features like context) although once built it
can target Go 1.4+.

Similarly to the instructions above, you'll need a python interpreter.
Having PyPy, python2 and python3 installed will allow you to build
all the possible engines & therefore packages etc, but just python2
is enough for the build to succeed, as long as you have cffi installed
as mentioned above.

You'll need to have dependencies for the various helper programs
of Please installed in order to build it. At the moment the minimal
set are Python (which you'll likely have anyway) and Java 7 or above.
Optional dependencies for various tests include unittest++
(`sudo apt-get install libunittest++-dev`), clang, gold and docker - none
of those are required to build components so their tests will be excluded
if they aren't available.

If you'd rather not worry about installing the dependencies, we provide
a prebuilt Docker image based on Ubuntu which is capable of building
the whole thing for you:
[`docker run -it thoughtmachine/please`](https://hub.docker.com/r/thoughtmachine/please)


Contributors
============

The core contributors so far are:
 * [Peter Ebden](https://github.com/peterebden)
 * [Will Montgomery](https://github.com/csdigi)
 * [Fabian Siddiqi](https://github.com/FS89)
 * [Diana-Maria Costea](https://github.com/dianacostea)
 * [Dimitar Pavlov](https://github.com/dimpavloff)

Progress has been slightly hindered by our fearless leader Paul
who continually insists that we should "get on with our work" and
"stop messing around with the build system". But he's not actually
fired us for spending time on this which he has our thanks for.
