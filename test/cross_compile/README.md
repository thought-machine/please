Example rules for testing cross-compiling.

One can test a simple binary with `plz build -a linux_x86 //test/cross_compile:bin`.
The configuration for that is stored in `.plzconfig_linux_x86` at the repo root.
This typically requires some additional packages to be installed; e.g. `gcc-multilib` or similar.
