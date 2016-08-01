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
curl -s https://get.plz.build | bash
```
In order for it to run you will need a Python interpreter available.
You can use either PyPy or CPython with cffi. On OSX PyPy is fairly
straightforward with Homebrew, simply run `brew install pypy`. You may
need to link the headers into /usr/local/include/pypy, at time of writing
Homebrew doesn't seem to do this automatically.

Unfortunately at the time of writing the Ubuntu and Debian PyPy packages
don't have the shared libraries. For now we suggest that you use the
packages from http://pypy.org.

Alternatively, you've likely got CPython installed already, and installing
cffi is as simple as `sudo pip install "cffi>=1.5.0"`. You can also use
apt-get but be sure you are getting a sufficiently recent version, older
distros (e.g. Trusty) may not package something new enough.

Then you simply run `plz init` at the root of your project to set up
a default config and you're good to start adding BUILD files.
See [the website](https://please.build) for more instructions about
how to write them.


Building Please
===============

To build Please yourself, run `./bootstrap.sh` in the repo root.
This will set up the minimal environment needed to build Please,
build it once manually and then rebuild it again using itself.
You'll need to have Go installed to build Please. It should
compile under Go 1.5+ (1.4 is no longer supported because we use
several minor standard library features from 1.5) and once compiled
can target 1.4+.

Similarly to the instructions above, you'll need a python interpreter.
Having PyPy, python2 and python3 installed will allow you to build
all the possible engines & therefore packages etc, but just python2
is enough for the build to succeed.

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
