// please_contain is a very small binary to implement sandboxing
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
#include <errno.h>
#include <stdlib.h>
#include <sched.h>
#include <string.h>
#include <sys/prctl.h>
#include <sys/types.h>
#include <sys/wait.h>

// drop_root is ported more or less directly from Chrome's chrome-sandbox helper.
//
static int drop_root() {
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

// exec_child calls execvp(3) to run the eventual child process.
int exec_child(char* argv[]) {
    return drop_root() ? 1 : execvp(argv[0], argv);
}

// clone_and_contain calls clone(2) to isolate and contain the network.
int clone_and_contain(char* argv[]) {
    const int STACK_SIZE = 10 * 1024;
    char* child_stack = malloc(STACK_SIZE);
    if (child_stack == NULL) {
        exit(3);
    }
    char* stack_top = child_stack + STACK_SIZE;  // assume stack grows downwards
    pid_t child_pid = clone((int (*)(void *))exec_child, stack_top, CLONE_NEWPID | CLONE_NEWNET | SIGCHLD, argv);
    if (child_pid == -1) {
        const int error = errno;
        fputs("failed to clone: ", stderr);
        fputs(strerror(error), stderr);
        fputs("\n", stderr);
        return error;
    }
    int exit_code = 0;
    if (waitpid(child_pid, &exit_code, 0) == -1) {
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
        fputs("please_contain implements sandboxing for Please.\n", stderr);
        fputs("It takes no flags, it simply executes the command given as arguments.\n", stderr);
        fputs("Usage: plz_contain command args...\n", stderr);
        exit(1);
    }
    return clone_and_contain(&argv[1]);
}
