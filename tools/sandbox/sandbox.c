#include "tools/sandbox/sandbox.h"

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#ifdef __linux__

#include <errno.h>
#include <sched.h>
#include <signal.h>
#include <net/if.h>
#include <net/route.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <sys/mount.h>
#include <sys/prctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <linux/rtnetlink.h>
#include <arpa/inet.h>

int perror_sock(char *errmsg, const int sock) {
    close(sock);
    perror(errmsg);
    return 1;
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
        perror_sock("SIOCGIFFLAGS", sock);
        return 1;
    }

    req.ifr_flags |= IFF_UP;
    if (ioctl(sock, SIOCSIFFLAGS, &req) < 0) {
        perror_sock("SIOCSIFFLAGS", sock);
        return 1;
    }
    close(sock);
    return 0;
}

// default_gateway adds a routing table entry for all traffic to be
// routed via the localhost. This is required to communicate with
// additional IP addresses added to the loopback interface that are
// outside of the 127.0.0.0/8 range.
int default_gateway() {
    const int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        perror("socket");
        return 1;
    }

    struct rtentry rte;
    struct sockaddr_in *sa;
    memset(&rte, 0, sizeof(rte));
    rte.rt_flags = RTF_UP | RTF_GATEWAY;

    sa = (struct sockaddr_in*) &rte.rt_gateway;
    sa->sin_family = AF_INET;
    sa->sin_addr.s_addr = inet_addr("127.0.0.1");

    sa = (struct sockaddr_in*) &rte.rt_dst;
    sa->sin_family = AF_INET;
    sa->sin_addr.s_addr = INADDR_ANY;

    sa = (struct sockaddr_in*) &rte.rt_genmask;
    sa->sin_family = AF_INET;
    sa->sin_addr.s_addr = INADDR_ANY;

    if (ioctl(sock, SIOCADDRT, &rte) < 0) {
        return perror_sock("SIOCADDRT", sock);
    }

    close(sock);
    return 0;
}

// add_local_ip assigns an additional IP address to the loopback interface.
// This is required for envtest to run in the sandbox which has a default
// cluster IP range of 10.0.0.0/24 and cannot use addresses in the local
// 127.0.0.0/8 range
int add_local_ip()
{
    const int sock = socket(AF_NETLINK, SOCK_RAW, NETLINK_ROUTE);
    if (sock < 0) {
        perror("socket");
        return 1;
    }

    struct sockaddr_nl sa;
    memset(&sa, 0, sizeof(sa));
    sa.nl_family = AF_NETLINK;

    if (bind(sock, (struct sockaddr*) &sa, sizeof(sa)) < 0) {
        return perror_sock("bind", sock);
    }

    struct {
        struct nlmsghdr nh;
        struct ifaddrmsg ifa;
        struct rtattr rta;
        in_addr_t addr;
    } req;
    memset(&req, 0, sizeof(req));

    req.nh.nlmsg_type = RTM_NEWADDR;
    req.nh.nlmsg_flags = NLM_F_CREATE | NLM_F_EXCL | NLM_F_REQUEST;
    req.nh.nlmsg_len = sizeof(req);

    req.ifa.ifa_family = AF_INET;
    req.ifa.ifa_prefixlen = 8;
    req.ifa.ifa_index = 1 ; // 1 is the loopback interface in our sandbox

    req.rta.rta_type = IFA_LOCAL;
    req.rta.rta_len = RTA_LENGTH(sizeof(req.addr));
    req.addr = inet_addr("10.1.1.1");

    if (send(sock, &req, req.nh.nlmsg_len, 0) < 0) {
        return perror_sock("send", sock);
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
// If the given string pointer (the argv[0] of the new process) is within the old temp dir
// then it will be replaced with a new version pointing into the new sandbox dir.
int mount_tmp(char** argv0) {
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
    // If SANDBOX_DIRS is set, we expect a comma-separated list of directories to mount a tmpfs over in order to hide them.
    // If one or more directories don't exist, that is OK, but any other error is fatal.
    char* dirs = getenv("SANDBOX_DIRS");
    if (dirs != NULL) {
      char *token = strtok(dirs, ",");
      while(token) {
        if (mount("tmpfs", token, "tmpfs", flags | MS_RDONLY, NULL) != 0) {
          if (errno == ENOENT || errno == ENOTDIR) {
            // This isn't fatal, it's OK for them not to exist (in that case we just have nothing to sandbox).
            fprintf(stderr, "Not mounting over %s since it isn't a directory\n", token);
          } else {
            perror("mount tmpfs");
            return 1;
          }
        }
        token = strtok(NULL, ",");
      }
      // Remove the env var; downstream things don't need to know what these were.
      unsetenv("SANDBOX_DIRS");
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
    change_env_vars(environ, dir, d);
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
    *argv0 = exec_name(*argv0, dir, d);
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
    if (mount_tmp(&arg->argv[0]) != 0) {
      return 1;
    }
    if (mount_proc() != 0) {
      return 1;
    }
  }
  if (arg->net) {
    if (lo_up() || add_local_ip() || default_gateway()) {
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

// exec_name returns the name of the new binary to exec() as.
// old_name is the current name; if it's within old_dir it will be re-prefixed to new_dir.
char* exec_name(const char* old_name, const char* old_dir, const char* new_dir) {
  return change_path(old_name, old_dir, new_dir, 0);
}

// change_path takes a string or environment variable and changes a prefix from one path to another.
char* change_path(const char* old_name, const char* old_dir, const char* new_dir, int prefix_len) {
  const int new_dir_len = strlen(new_dir);
  const int old_dir_len = strlen(old_dir);
  const int old_name_len = strlen(old_name);
  if (strncmp(old_dir, old_name + prefix_len, old_dir_len) != 0) {  // is the value of old_name prefixed with old_dir
    return (char*)old_name;  // Dodgy cast but we know we don't alter it again later.
  }
  const int new_len = new_dir_len + old_name_len - old_dir_len + 1;
  char* new_name = malloc(new_len + 1);
  strncpy(new_name, old_name, prefix_len);
  strcpy(new_name + prefix_len, new_dir);
  strcpy(new_name + prefix_len + new_dir_len, old_name + prefix_len + old_dir_len);
  new_name[new_len] = 0;
  return new_name;
}

// change_env_vars changes any environment variables prefixed with the old directory to the new one.
void change_env_vars(char** environ, const char* old_dir, const char* new_dir) {
  for (char** env = environ; *env; ++env) {
    const char* equals = strchr(*env, '=');
    if (equals) {
      *env = change_path(*env, old_dir, new_dir, equals - *env + 1);
    }
  }
}
