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
#include <unistd.h>

#ifdef __linux__
#include <stdlib.h>
#include <sched.h>
#include <string.h>
#include <net/if.h>
#include <sys/ioctl.h>
#include <sys/prctl.h>
#include <sys/types.h>
#include <sys/wait.h>

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

// exec_child calls execvp(3) to run the eventual child process.
int exec_child(char* argv[]) {
    if (lo_up() != 0) {
        return 1;
    }
    if (drop_root() != 0) {
        return 1;
    }
    return execvp(argv[0], argv);
}

// clone_and_contain calls clone(2) to isolate and contain the network.
int clone_and_contain(char* argv[]) {
    // Unsure exactly how big this needs to be, but 20k is probably enough
    // given that we don't do an awful lot in the child process.
    const int STACK_SIZE = 20 * 1024;
    char* child_stack = malloc(STACK_SIZE);
    if (child_stack == NULL) {
        perror("malloc");
        exit(3);
    }
    char* stack_top = child_stack + STACK_SIZE;  // assume stack grows downwards
    pid_t child_pid = clone((int (*)(void *))exec_child,
                            stack_top,
                            CLONE_NEWPID | CLONE_NEWNET | CLONE_NEWIPC | CLONE_NEWUTS | SIGCHLD,
                            argv);
    if (child_pid == -1) {
        perror("clone");
        return -1;
    }
    int exit_code = 0;
    if (waitpid(child_pid, &exit_code, 0) == -1) {
        perror("waitpid");
        return -1;
    }
    return WEXITSTATUS(exit_code);
}

#else

// On non-Linux systems clone_and_contain simply execs a subprocess.
// It's not really expected to be used there, this is simply to make it compile.
int clone_and_contain(char* argv[]) {
    return execvp(argv[0], argv);
}

#endif


int main(int argc, char* argv[]) {
    if (argc < 2) {
        fputs("please_sandbox implements sandboxing for Please.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: plz_contain command args...\n", stderr);
        exit(1);
    }
    return clone_and_contain(&argv[1]);
}
