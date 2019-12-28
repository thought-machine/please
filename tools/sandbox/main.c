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
#include <sys/stat.h>
#include <sys/types.h>

// Support Linux versions < 4.0ish that don't have MS_LAZYTIME
#ifndef MS_LAZYTIME
#define MS_LAZYTIME	(1<<25)
#endif

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

// deny_groups disables the ability to call setgroups(2). This is required
// before we can successfully write to gid_map in map_ids.
int deny_groups() {
    FILE* f = fopen("/proc/self/setgroups", "w");
    if (!f) {
        perror("fopen /proc/self/setgroups");
        return 1;
    }
    if (fputs("deny\n", f) < 0) {
        perror("fputs");
        return 1;
    }
    return fclose(f);
}

// map_ids maps the user id or group id inside the namespace to those outside.
// Without this we fail to create directories in the tmpfs with an EOVERFLOW.
int map_ids(int out_id, const char* path) {
    FILE* f = fopen(path, "w");
    if (!f) {
        perror("fopen");
        return 1;
    }
    if (fprintf(f, "%d %d 1\n", out_id, out_id) < 0) {
        perror("fprintf");
        return 1;
    }
    if (fclose(f) != 0) {
        perror("fclose");
        return 1;
    }
    return 0;
}

// mount_tmp mounts a tmpfs on /tmp for the tests to muck about in and
// bind mounts the test directory to /tmp/plz_sandbox.
int mount_tmp() {
    // Don't mount on /tmp if our tmp dir is under there, otherwise we won't be able to see it.
    const char* dir = getenv("TMP_DIR");
    const char* d = "/tmp/plz_sandbox";
    if (dir) {
        if (strncmp(dir, "/tmp/", 5) == 0) {
            fputs("Not mounting tmpfs on /tmp since TMP_DIR is a subdir\n", stderr);
            return 0;
        }
    }
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
    if (setenv("TMPDIR", "/tmp", 1) != 0) {
        perror("setenv");
        return 1;
    }
    if (!dir) {
        fputs("TMP_DIR not set, will not bind-mount to /tmp/plz_sandbox\n", stderr);
        return 0;
    }
    if (mkdir(d, S_IRWXU) != 0) {
        perror("mkdir /tmp/plz_sandbox");
        return 1;
    }
    if (mount(dir, d, "", MS_BIND, NULL) != 0) {
        perror("bind mount");
        return 1;
    }
    if (setenv("TEST_DIR", d, 1) != 0 ||
        setenv("TMP_DIR", d, 1) != 0 ||
        setenv("HOME", d, 1) != 0) {
        perror("setenv");
        return 1;
    }
    return chdir(d);
}

// contain separates the process into new namespaces to sandbox it.
int contain(char* argv[]) {
    const uid_t uid = getuid();
    const uid_t gid = getgid();
    if (unshare(CLONE_NEWUSER | CLONE_NEWNET | CLONE_NEWUTS | CLONE_NEWIPC | CLONE_NEWNS | CLONE_NEWNET) != 0) {
        perror("unshare");
        fputs("Your user doesn't seem to have enough permissions to call unshare(2).\n", stderr);
        fputs("please_sandbox requires support for user namespaces (usually >= Linux 3.10)\n", stderr);
        return 1;
    }
    if (deny_groups() != 0) {
      return 1;
    }
    if (map_ids(uid, "/proc/self/uid_map") != 0 ||
        map_ids(gid, "/proc/self/gid_map") != 0) {
        return 1;
    }
    if (mount_tmp() != 0) {
        return 1;
    }
    if (lo_up() != 0) {
      return 1;
    }
    if (execvp(argv[0], argv) != 0) {
        fprintf(stderr, "exec %s: ", argv[0]);
        perror("");
        return 1;
    }
    return 0;
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
