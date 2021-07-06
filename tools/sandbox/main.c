// please_sandbox is a very small binary to implement sandboxing
// of tests (and possibly other build actions) via cgroups.
// Essentially this is a very lightweight replacement for Docker
// where we would use it for tests to avoid port clashes etc.
//
// Note that this is a no-op on non-Linux OSs because they will not
// support namespaces / cgroups. We still behave similarly otherwise
// in order for it to be transparent to the rest of the system.
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "tools/sandbox/sandbox.h"

int main(int argc, char* argv[]) {
    if (argc < 2) {
        fputs("please_sandbox implements sandboxing for Please.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: please_sandbox command args...\n", stderr);
        return 1;
    }

    // Network namespace is sandboxed by default unless the `SANDBOX_NETWORK=0` env exists 
    const char* sandbox_network_env = getenv("SANDBOX_NETWORK");
    const bool sandbox_network = sandbox_network_env == NULL || strcmp(sandbox_network_env, "0");

    // Mount namespace is sandboxed by default unless the `SANDBOX_MOUNT=0` env exists 
    const char* sandbox_mount_env = getenv("SANDBOX_MOUNT");
    const bool sandbox_mount = sandbox_mount_env == NULL || strcmp(sandbox_network_env, "0");

    return contain(&argv[1], sandbox_network, sandbox_mount);
}
