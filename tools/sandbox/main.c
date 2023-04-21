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
    int flags = 0;

    if (argc < 2) {
        fputs("please_sandbox implements sandboxing for Please.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: please_sandbox command args...\n", stderr);
        return 1;
    }

    // Network namespace is sandboxed by default but it can be opted out if `SHARE_NETWORK=1` env is set
    const char* share_network_env = getenv("SHARE_NETWORK");
    if (share_network_env == NULL || !strcmp(share_network_env, "1")) {
        flags |= FLAG_SANDBOX_NET;
    }

    // Mount namespace is sandboxed by default but it can be opted out if `SHARE_MOUNT=1` env is set
    const char* share_mount_env = getenv("SHARE_MOUNT");
    if (share_mount_env == NULL || !strcmp(share_mount_env, "1")) {
        flags |= FLAG_SANDBOX_FS;
    }

    return contain(&argv[1], flags);
}
