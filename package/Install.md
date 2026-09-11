# Installing Please on Windows

This is the Windows build of Please. Windows support is new, so read the last two sections
before you rely on it.

## Install

1. Extract the zip. It contains a single `please` directory; put that wherever you keep tools,
   for example `C:\Tools\please`. Nothing writes to the directory afterwards, so Program Files
   is fine too.
2. Add that directory to your `PATH`, so that `plz` works from any repository.
3. Check it:

   ```
   plz --version
   ```

`plz.cmd` is the entry point, and it does nothing but run `please.exe` beside it. Everywhere
else Please installs `plz` as a symlink; Windows needs Developer Mode for those, so a one-line
batch file stands in.

There is no installer, no registry key and no service. Uninstalling is deleting the directory.

## What is in here

| File | What it is |
|---|---|
| `please.exe` | Please itself |
| `plz.cmd` | the short name you type |
| `busybox.exe` | the shell that build actions run in |
| `build_langserver.exe` | the BUILD-file language server, for editor integration |
| `arcat.exe` | Please's archive tool, used by many built-in rules |

Some builds also carry the language plugins, as `plugin_*.zip` alongside `please_go.exe`,
`please_cc.exe` and `please_pex.exe`. If `plugin_revisions.txt` is here, yours is one of them,
and that file says exactly which build of each plugin you have.

**Keep these files together.** Please finds the shell, the archive tool and the bundled plugins
by looking beside its own binary. Copying `please.exe` out on its own leaves it unable to run a
build action.

## Nothing else to configure

A repository that asks for the go, cc, shell or python plugin the ordinary way works as it is.
Where this build bundles them, they resolve from the install directory rather than being
downloaded, so a machine with no internet access builds the same as one with it. To use the
published plugins instead, delete the `plugin_*.zip` files.

**Language toolchains are not bundled, and never are on any platform.** Building Go needs Go,
C++ needs a compiler, and Python needs an interpreter, each installed separately and pointed at
from your repository's `.plzconfig` the same way it would be anywhere else. What is bundled is
only what Please itself needs.

## Known limitations

- **`plz update` does not refresh everything.** It fetches only the Please binary, so busybox,
  the archive tool and any bundled plugins stay at the version you first installed. Re-extract
  the zip instead of updating in place.
- **A repository that pins an exact `[please] version`** will try to download that version from
  the Please download server, which has no Windows release yet. Use a `>=` constraint, or set
  `selfupdate = false`.
- **Build sandboxing is off.** It is built on Linux namespaces and there is no Windows
  equivalent yet, so build actions and tests see the whole machine.
- **Real-time antivirus scanning locks files Please has just written**, which shows up as
  intermittent sharing-violation errors and slow builds. Excluding your `plz-out` directories
  helps.
- **Long paths.** Output paths nest deeply. If your repository lives far from the root of a
  drive, turn on Windows long-path support.

## Where a bundled plugin came from

If this build carries plugins, `plugin_revisions.txt` names the exact commit of each, and they
are **not** the upstream releases their version numbers suggest: each is that release plus
Windows fixes that are not published anywhere yet.

They also answer for whatever revision your repository asks for. A repository pinning a
different version of a plugin silently gets the bundled one on Windows. That is what lets an
unmodified repository build with no network at all; deleting the archive opts back out of it.
