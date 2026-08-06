# please_sandbox

> [!CAUTION]
> The Please Sandbox is not a security boundary. It is not designed to run untrusted or malicious
> code.

`please_sandbox` is a wrapper that allows running a given binary in Linux namespaces. By default, it
creates PID, IPC, UTS, user, mount and network namespaces. It also does a bunch of things on the
filesystem:

- if `TMP_DIR` is not set or not under `/tmp`, a tmpfs is mounted over `/tmp` and `TMPDIR` is set to
  `/tmp`. If it is set, `$TMP_DIR` is bind mounted onto `/tmp/plz_sandbox`, which becomes the
  working directory, and the root filesystem is remounted read-only;
- if `SANDBOX_DIRS` is set, we expect a comma-separated list of path that will be hidden with a
  tmpfs;
- if `SANDBOX_FILE_MOUNTS` is set, we expect it to be set to a comma-separated list of key-value
  pairs in the following format: `key:value`. Keys must point to existing paths and will be bind
  mounted to the path given as value;
- if `SANDBOX_UID_MAP` and `SANDBOX_GID_MAP` are set (both are required if either is), we pass
  these arguments to newuidmap/newgidmap to configure uid/gid mappings. The format is 1..n
  space-delimited triples of [id lowerid count], see `man newuidmap`;

Mount and network namespaces can be disabled setting the `SHARE_MOUNT` and `SHARE_NETWORK`
environment variables to `1`. Remounting of `/proc` can be disabled by setting `MOUNT_PROC=0`,
e.g. for systems that don't allow mounting a full `/proc` from a new user namespace (see the
mount namespace notes below).

When the network namespace is used, the loopback interface is brought up with an additional IP
address. `SANDBOX_LOCAL_IP` defines, defined empty string disables the extra address.

## Using the knobs with Please

Please invokes the tool configured in `[sandbox] tool` with the command to run as its arguments,
and sets `SHARE_NETWORK` and `SHARE_MOUNT` itself according to the rule being run. Remote
execution workers invoke the sandbox tool themselves and should set these variables explicitly,
since the binary's defaults apply when they are absent. To set the other knobs, or to override
Please's per-rule choices, point `tool` at a thin wrapper script:

```sh
#!/bin/sh
# please_sandbox, but without remounting /proc.
export MOUNT_PROC=0
exec /path/to/please_sandbox "$@"
```

```ini
[sandbox]
tool = /path/to/noproc_sandbox_wrapper
```

## UID/GID mapping in user namespace

By default, the sandbox only maps the real UID/GID of the user running the sandbox into the namespace.
However, `SANDBOX_UID_MAP` and `SANDBOX_GID_MAP` may be used to define arguments that are passed to
`new*idmap`.

The example below will map the real UID of the user running the sandbox to root and UIDs
from range [100000;165536) to [1;65536) in the child namespace (assuming your UID is 1000):

```bash
$ TMP_DIR=/tmp SANDBOX_UID_MAP="0 $UID 1  1 100000 65536" SANDBOX_GID_MAP="0 $(id -g) 1" please_sandbox cat /proc/self/uid_map
         0       1000          1
         1     100000      65536
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
