// noproc_sandbox is a slightly modified version of please_sandbox that does all the same
// things except it doesn't mount /proc.
// This is a specific, if hacky, solution for newer versions of systemd which aren't
// allowing us to mount a full /proc from a new user namespace.
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "tools/sandbox/sandbox.h"

int main(int argc, char* argv[]) {
    if (argc < 2) {
        fputs("noproc_sandbox implements limited sandboxing via Linux namespaces.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: noproc_sandbox command args...\n", stderr);
        return 1;
    }

    // Network namespace is sandboxed by default but it can be opted out if `SHARE_NETWORK=1` env is set
    const char* share_network_env = getenv("SHARE_NETWORK");
    const bool unshare_network = share_network_env == NULL || strcmp(share_network_env, "1");

    // Mount namespace is sandboxed by default but it can be opted out if `SHARE_MOUNT=1` env is set
    const char* share_mount_env = getenv("SHARE_MOUNT");
    const bool unshare_mount = share_mount_env == NULL || strcmp(share_mount_env, "1");

    return contain(&argv[1], unshare_network, unshare_mount, unshare_mount, false);
}
