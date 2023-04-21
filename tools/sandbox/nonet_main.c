// nonet_sandbox is a slightly modified version of please_sandbox that does all the same
// things except it leaves the network unscathed.
// It is currently not used, but is conceptually useful to sandbox rules that request sandbox
// disabling in order to gain network access (which is by far the most common case for that),
// but it's still useful to contain the other namespaces.
#include <stdio.h>
#include "tools/sandbox/sandbox.h"

int main(int argc, char* argv[]) {
    if (argc < 2) {
        fputs("nonet_sandbox implements limited sandboxing via Linux namespaces.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: nonet_sandbox command args...\n", stderr);
        return 1;
    }
    return contain(&argv[1], FLAG_SANDBOX_ALL & ~FLAG_SANDBOX_NET);
}
