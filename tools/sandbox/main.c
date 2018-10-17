// please_sandbox is a very small binary to implement sandboxing
// of tests (and possibly other build actions) via cgroups.
// Essentially this is a very lightweight replacement for Docker
// where we would use it for tests to avoid port clashes etc.
//
// Note that this is a no-op on non-Linux OSs because they will not
// support namespaces / cgroups. We still behave similarly otherwise
// in order for it to be transparent to the rest of the system.

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#ifdef __linux__
#include <sched.h>
#include <string.h>
#include <net/if.h>
#include <sys/ioctl.h>
#include <sys/mount.h>
#include <sys/prctl.h>
#include <sys/types.h>

// TODO(peterebden): Remove the following once our build machine gets updated...
#ifndef MS_LAZYTIME
#define MS_LAZYTIME	(1<<25)
#endif

// drop_root is ported more or less directly from Chrome's chrome-sandbox helper.
// It simply drops us back to whatever user invoked us originally (i.e. before suid
// got involved).
int drop_root() {
    if (prctl(PR_SET_DUMPABLE, 0, 0, 0, 0)) {
        perror("prctl(PR_SET_DUMPABLE)");
        return 1;
    }
    if (prctl(PR_GET_DUMPABLE, 0, 0, 0, 0)) {
        perror("Still dumpable after prctl(PR_SET_DUMPABLE)");
        return 1;
    }
    gid_t rgid, egid, sgid;
    if (getresgid(&rgid, &egid, &sgid)) {
        perror("getresgid");
        return 1;
    }
    if (setresgid(rgid, rgid, rgid)) {
        perror("setresgid");
        return 1;
    }
    uid_t ruid, euid, suid;
    if (getresuid(&ruid, &euid, &suid)) {
        perror("getresuid");
        return 1;
    }
    if (setresuid(ruid, ruid, ruid)) {
        perror("setresuid");
        return 1;
    }
    return 0;
}

// lo_up brings up the loopback interface in the new network namespace.
// By default the namespace is created with lo but it is down.
// Note that this can't be done with system() because it loses the
// required capabilities.
int lo_up() {
    const int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        perror("socket");
        return 1;
    }

    struct ifreq req;
    memset(&req, 0, sizeof(req));
    strncpy(req.ifr_name, "lo", IFNAMSIZ);
    if (ioctl(sock, SIOCGIFFLAGS, &req) < 0) {
        perror("SIOCGIFFLAGS");
        return 1;
    }

    req.ifr_flags |= IFF_UP;
    if (ioctl(sock, SIOCSIFFLAGS, &req) < 0) {
        perror("SIOCSIFFLAGS");
        return 1;
    }
    close(sock);
    return 0;
}

// mount_tmp mounts a tmpfs on /tmp for the tests to muck about in.
int mount_tmp() {
    // Remounting / as private is necessary so that the tmpfs mount isn't visible to anyone else.
    if (mount("none", "/", NULL, MS_REC | MS_PRIVATE, NULL) != 0) {
        perror("remount");
        return 1;
    }
    const int flags = MS_LAZYTIME | MS_NOATIME | MS_NODEV | MS_NOSUID;
    if (mount("tmpfs", "/tmp", "tmpfs", flags, NULL) != 0) {
        perror("mount");
        return 1;
    }
    return setenv("TMPDIR", "/tmp", 1);
}

// contain separates the process into new namespaces to sandbox it.
int contain(char* argv[]) {
    if (unshare(CLONE_NEWNET | CLONE_NEWUTS | CLONE_NEWIPC | CLONE_NEWNS) != 0) {
        perror("unshare");
        fputs("Your user doesn't seem to have enough permissions to call unshare(2).\n", stderr);
        fputs("plz_sandbox normally needs to be installed setuid root in order to work.\n", stderr);
        return 1;
    }
    if (mount_tmp() != 0) {
      return 1;
    }
    if (lo_up() != 0) {
        return 1;
    }
    if (drop_root() != 0) {
        return 1;
    }
    return execvp(argv[0], argv);
}

#else

// On non-Linux systems contain simply execs a subprocess.
// It's not really expected to be used there, this is simply to make it compile.
int contain(char* argv[]) {
    return execvp(argv[0], argv);
}

#endif


int main(int argc, char* argv[]) {
    if (argc < 2) {
        fputs("please_sandbox implements sandboxing for Please.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: plz_sandbox command args...\n", stderr);
        exit(1);
    }
    return contain(&argv[1]);
}
