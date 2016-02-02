Please is a gentleman's build tool.

It's heavily inspired by Blaze/Bazel and Buck and follows a lot of
similar concepts, but aims to be lighter weight and more flexible
in its build rules.

See http://please.build for more information.

TODO(all): Set up the website.


Getting Started
===============

To get started with Please, run ./bootstrap.sh in the repo root.
This will set up the minimal environment needed to build Please,
build it once manually and then rebuild it again using itself.
You'll need to have Go 1.5 and PyPy installed to use Please.
If you're using Ubuntu, run sudo apt-get install golang to install
Go.

Unfortunately at the time of writing the Ubuntu PyPy packages don't
include any shared libraries. We've stashed a .deb at
https://s3-eu-west-1.amazonaws.com/please-build/pypy_4.0.0_amd64.deb
which contains what you need.

To build on OSX, you'll need Homebrew installed. After that it's
rather easier; simply 'brew install pypy' and 'brew install go'
and you should be good to go (heh).

You'll need to have dependencies for the various helper programs
of Please installed in order to build it. At the moment the minimal
set are Python (which you'll likely have anyway) and Java 7 or above.
Optional dependencies include unittest++
(sudo apt-get install libunittest++-dev), clang, gold and docker - none
of those are required to build components so their tests will be excluded
if they aren't available.


TODOs / Future development
==========================

At time of writing, Please has passed 8000 lines of Go, and 1000 lines
of Python for the built-in rules. I'm reasonably happy with those (albeit
it's a little more than I originally planned on...) but always keen to
do a little gentle refactoring.
Hopefully it's going to stay at about the same size, at least as an
order of magnitude.

More tests would be nice. I've been a bit slack, mostly because a lot
of parts of Please are a pain to test because it has so much filesystem
interaction.

Real multithreaded parsing: since we moved to PyPy we can now parse on any
thread (the original implementation used CPython and couldn't).
Unfortunately it's still limited by the GIL, it would be cool if we could
parse on many threads truly simultaneously. I'm keeping a surreptitious
eye on the pypy-stm branch for this but it's probably a long way off still.

Customise the build language more; currently it's straight Python with
many of the builtins and a small number of statements banned. Would be cool
to have a more heavily customised dialect to help encourage correctness;
an obvious candidate here would be enforcing ordering on all dicts to
avoid indeterminacy of rules that iterate them, or the (admittedly minor)
performance penalty of sorting them before iteration.

Deprecate include_defs in favour of subinclude since it can't be
controlled by visibility attributes. Want to be sure subinclude doesn't
compromise significantly on efficiency before we do though.
