Please is a cross-language build system with an emphasis on
high performance, extensibility and reproduceability.
It supports a number of popular languages and can automate
nearly any aspect of your build process.

See http://please.build for more information.

TODO(all): Set up the website.


Getting Started
===============

The easiest way to install it on your own machine is to run:
```bash
curl -s https://s3-eu-west-1.amazonaws.com/please-build/get_plz.sh | bash
```
You will need to have PyPy installed for it to run. On OSX this is
straightforward with Homebrew, simply run `brew install pypy`.

Unfortunately at the time of writing the Ubuntu and Debian PyPy packages
don't have the shared libraries. For now we suggest that you use the
packages from http://pypy.org.

Then you simply run `plz init` at the root of your project to set up
a default config and you're good to start adding BUILD files.
See [the website](https://please.build) for more instructions about
how to write them.


Building Please
===============

To build Please yourself, run `./bootstrap.sh` in the repo root.
This will set up the minimal environment needed to build Please,
build it once manually and then rebuild it again using itself.
You'll need to have Go 1.6 and PyPy installed to build Please.
Go 1.5 and earlier are not currently supported due to subtle cgo
incompatibilities (Note that this is only for building Please itself;
it can target Go 1.4 or later).

To build on OSX, you'll need Homebrew installed. After that simply
'brew install pypy' and 'brew install go' and you should be good to go (heh).

You'll need to have dependencies for the various helper programs
of Please installed in order to build it. At the moment the minimal
set are Python (which you'll likely have anyway) and Java 7 or above.
Optional dependencies for various tests include unittest++
(`sudo apt-get install libunittest++-dev`), clang, gold and docker - none
of those are required to build components so their tests will be excluded
if they aren't available.


Contributors
============

The core contributors so far are:
 * [Peter Ebden](https://github.com/peterebden)
 * [Will Montgomery](https://github.com/csdigi)
 * [Fabian Siddiqi](https://github.com/FS89)
 * [Diana-Maria Costea](https://github.com/dianacostea)

Progress has been slightly hindered by our fearless leader Paul
who continually insists that we should "get on with our work" and
"stop messing around with the build system". But he's not actually
fired us for spending time on this which he has our thanks for.