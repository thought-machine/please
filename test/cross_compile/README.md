Example rules for testing cross-compiling.

One can test a simple binary with `plz build -a linux_arm64 //test/cross_compile:bin`.

These use Go rather than C deliberately: the Go toolchain can produce binaries for any platform
it supports without anything extra being installed, whereas the C version of this needed a
cross-compiling gcc (e.g. `gcc-multilib`) which is often not present.

The `package()` call in the BUILD file turns off static linking for these rules; a static link
runs the host C compiler, which can't handle objects for the architecture we're targeting.
