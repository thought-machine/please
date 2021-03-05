#include "tools/sandbox/sandbox.h"

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#ifdef __linux__

#include <sched.h>
#include <signal.h>
#include <string.h>
#include <net/if.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <sys/mount.h>
#include <sys/prctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>

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
    // Now make root readonly (once we have bind-mounted in the non-readonly workdir)
    if (mount("none", "/", NULL, MS_REMOUNT | MS_RDONLY | MS_BIND, NULL) != 0) {
        perror("remount ro");
        return 1;
    }
    return chdir(d);
}

// mount_tmpfs mounts an empty tmpfs at the given location.
int mount_tmpfs(const char* dir) {
    const int flags = MS_LAZYTIME | MS_NOATIME | MS_NODEV | MS_NOSUID | MS_NOEXEC;
    if (mount("tmpfs", dir, "tmpfs", flags, NULL) != 0) {
        perror("mount tmpfs");
        return 1;
    }
    return 0;
}

// mount_proc mounts a new procfs on /proc
int mount_proc() {
  if (mount("proc", "/proc", "proc", 0, NULL) != 0) {
    perror("mount proc");
    return 1;
  }
  return 0;
}

typedef struct _clone_arg {
  uid_t uid;
  uid_t gid;
  bool  net;
  bool  mount;
  char** argv;
} clone_arg;

// contain_child is the entrypoint for the child process.
int contain_child(void* p) {
  clone_arg* arg = p;
  if (deny_groups() != 0) {
    return 1;
  }
  if (map_ids(arg->uid, "/proc/self/uid_map") != 0 ||
      map_ids(arg->gid, "/proc/self/gid_map") != 0) {
    return 1;
  }
  if (arg->mount) {
    if (mount_tmp() != 0) {
      return 1;
    }
    if (mount_proc() != 0) {
      return 1;
    }
  }
  if (arg->net) {
    if (lo_up() != 0) {
      return 1;
    }
  }
  if (prctl(PR_SET_PDEATHSIG, SIGKILL) == -1) {
    perror("failed to set PDEATHSIG");
    return 1;
  }
  execvp(arg->argv[0], arg->argv);  // If this returns, an error has occurred.
  fprintf(stderr, "exec %s: ", arg->argv[0]);
  perror("");
  return 1;
}

// contain separates the process into new namespaces to sandbox it.
int contain(char* argv[], bool net, bool mount) {
  clone_arg arg;
  arg.uid = getuid();
  arg.gid = getgid();
  arg.argv = argv;
  arg.net = net;
  arg.mount = mount;

  static const int stack_size = 100 * 1024;
  char* stack = mmap(NULL, stack_size, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS | MAP_STACK, -1, 0);
  if (stack == MAP_FAILED) {
    perror("mmap");
    return 1;
  }
  const int ns = CLONE_NEWUSER | CLONE_NEWUTS | CLONE_NEWIPC | CLONE_NEWPID | (net ? CLONE_NEWNET : 0) | (mount ? CLONE_NEWNS : 0);
  pid_t pid = clone(contain_child, stack + stack_size, ns | SIGCHLD, &arg);
  if (pid == -1) {
    perror("clone");
    fputs("Your user doesn't seem to have enough permissions to call clone(2).\n", stderr);
    fputs("please_sandbox requires support for user namespaces (usually >= Linux 3.10)\n", stderr);
    return 1;
  }
  // We're the parent process; wait on the child and exit with its status.
  int status = 0;
  if (waitpid(pid, &status, 0) == -1) {
    perror("waitpid failed");
  }
  if (WIFEXITED(status)) {
    return WEXITSTATUS(status);
  } else if (WIFSIGNALED(status)) {
    kill(getpid(), WTERMSIG(status));
  }
  fprintf(stderr, "child exit failed\n");
  return 1;
}

#else  // __linux__

// On non-Linux systems contain simply execs a subprocess.
// It's not really expected to be used there, this is simply to make it compile.
int contain(char* argv[], bool net, bool mount) {
  return execvp(argv[0], argv);
}

#endif  // __linux__
