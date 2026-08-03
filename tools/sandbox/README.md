# tm_sandbox

> [!CAUTION]
> The Please Sandbox is not a security boundary. It is not designed to run untrusted or malicious
> code.

`tm_sandbox` is a wrapper that allows running a given binary in Linux namespaces. By default, it
creates PID, IPC, UTS, user, mount and network namespaces. It also does a bunch of things on the
filesystem:

- if `TMP_DIR` is not set or not under `/tmp`, a tmpfs is mounted over `/tmp` and `TMPDIR` is set to
  `/tmp`. If it is set, current working directory is bind mounted to its path;
- if `SANDBOX_DIRS` is set, we expect a comma-separated list of path that will be hidden with a
  tmpfs;
- if `SANDBOX_FILE_MOUNTS` is set, we expect it to be set to a comma-separated list of key-value
  pairs in the following format: `key:value`. Keys must point to existing paths and will be bind
  mounted to the path given as value;
- if `SANDBOX_UID_MAP` or `SANDBOX_GID_MAP` is set, we pass these arguments to
  newuidmap/newgidmap to configure uid/gid mappings. The format is 1..n space-delimited triples
  of [id lowerid count], see `man newuidmap`;

Mount and network namespaces can be disabled setting the `SHARE_MOUNT` and `SHARE_NETWORK`
environment variables to `1`.

The sandbox is distributed with 3 other flavours:

- `nonet_sandbox` disables network namespacing by default, but it can be force-enabled by setting
  `SHARE_NETWORK` to 0.
- `noproc_sandbox` disables remounting of `/proc`.
- `nonetproc_sandbox` disables remounting of `/proc` and disables network namespacing by default.

## UID/GID mapping in user namespace

By default, the sandbox only maps the effective UID/GID from the running sandbox into the namespace.
However, `SANDBOX_UID_MAP` and `SANDBOX_GID_MAP` may be used to define arguments that are passed to
`new*idmap`.

The example below will map the effective UID/GID from the running sandbox to root and UIDs/GIDs
from range [100000;165536) to [1;65536) in the child namespace.

```bash
$ TMP_DIR=/tmp SANDBOX_UID_MAP="0 $UID 1  1 100000 65536" tm_sandbox cat /proc/self/uid_map
         0  100000000          1
         1    100000       65536
```

## Capabilities and other requirements

### Namespaces

Historically, creating Mount, PID, IPC, UTS, and Network namespaces required the heavily overloaded
`CAP_SYS_ADMIN` capability on the host system. However, since Linux 3.8, unprivileged processes can
use User Namespaces to obtain local `CAP_SYS_ADMIN` privileges. This allows processes to create
Mount, PID, IPC, UTS, and Network namespaces without needing host-level root or `CAP_SYS_ADMIN`
privileges.

#### Mount namespace

In addition to the above, when enabling the mount namespace, the sandbox will remount /proc, so any
process that use it will have access to accurate information of the PID namespace (otherwise they'd
still have access to /proc from the parent namespace). There's however a specific edge case in the
Linux kernel, that prevents /proc from being remounted in a mount namespace, when the parent /proc
is not fully visible, [see this commit](https://github.com/torvalds/linux/commit/1b852bceb0d1).

This is an issue with most container runtimes (and therefore Kubernetes), as by default, they will
hide some part of /proc in a container to reduce attack surface, eg
[see this docker PR](https://github.com/docker/cli/pull/1808).
