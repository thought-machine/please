#!/usr/bin/env bash
#
# Vendors the plugin sources and helper tools that the offline Windows release bundles, into
# third_party/plugins, which is gitignored. See docs/design/windows/08-offline-release.md.
#
# This exists because the Windows fixes for all four plugins are on branches nobody has
# published, so the revisions plugins/BUILD pins do not contain them. It is a workaround, and
# the whole of third_party/plugins goes away when those branches land upstream.
#
# Nothing here can be a build rule. A rule reading a checkout through its absolute path hashes
# the path rather than the tree, so an edited plugin would be served stale from the cache for
# ever; and the release is built on machines that have no checkouts at all. So: a script, run
# by hand, writing files that Please then hashes like any other source.
set -euo pipefail

cd "$(dirname "$0")/../.."
readonly OUT="third_party/plugins"
readonly BRANCH="windows"

# plugin name -> the helper tool it ships, if any. These are host tools: they run on the
# machine doing the building, so a Windows install needs Windows builds of them.
declare -A TOOLS=(
  [go-rules]=please_go
  [cc-rules]=please_cc
  [python-rules]=please_pex
  [shell-rules]=
)

# The Please to build the tools with. Ours, not whatever is on the PATH - the installed one is
# routinely older than this branch.
readonly PLZ="$PWD/plz-out/bin/src/please"

die() { echo "vendor_plugins: $*" >&2; exit 1; }

[ -x "$PLZ" ] || die "$PLZ is not built. Run 'plz build //src:please' first."

# Each checkout's path comes from the same [buildconfig] keys that .plzconfig.local uses to
# build against them, so there is one place to say where they are.
plugin_path() {
  "$PLZ" query config 2>/dev/null | sed -n "s|^$1-path = ||p" | tail -1
}

mkdir -p "$OUT"
: > "$OUT/plugin_revisions.txt"

cat >> "$OUT/plugin_revisions.txt" <<'HEADER'
The plugins bundled in this Please install.

These are NOT the upstream releases they are versioned against. Each is that tag plus the
Windows commits from a branch that is not merged anywhere.

The archive filenames carry no revision, so whatever revision a repo's plugin_repo() asks for
resolves to the copy here. A repo pinning a different version of a plugin gets this one instead
on Windows. That is what makes the install work with no network; delete an archive to opt out
of it for that plugin.

HEADER

for plugin in "${!TOOLS[@]}"; do
  path="$(plugin_path "$plugin")"
  [ -n "$path" ] && [ -d "$path" ] || die "no checkout for $plugin; set ${plugin}-path under [buildconfig]"

  branch="$(git -C "$path" rev-parse --abbrev-ref HEAD)"
  [ "$branch" = "$BRANCH" ] || die "$path is on $branch, not $BRANCH"
  # --porcelain rather than diff-index, whose stat cache goes stale after a build and reports
  # changes that are not there.
  [ -z "$(git -C "$path" status --porcelain --untracked-files=no)" ] ||
    die "$path has uncommitted changes"

  sha="$(git -C "$path" rev-parse --short HEAD)"
  described="$(git -C "$path" describe --tags 2>/dev/null || echo "$sha")"
  subject="$(git -C "$path" log -1 --format=%s)"

  # git archive rather than the working tree: the trees carry plz-out and .plzconfig.local, and
  # the commit is the only durable name these branches have. --prefix gives the single
  # top-level directory holding a .plzconfig that plugin_repo()'s extract step looks for.
  echo "vendoring $plugin at $sha"
  git -C "$path" archive --format=zip --prefix="$plugin-$sha/" HEAD > "$OUT/plugin_$plugin.zip"

  tool="${TOOLS[$plugin]}"
  if [ -n "$tool" ]; then
    # Built in the plugin's own repo rather than through ///go//tools/please_go and friends
    # from ours. Cross-compiling a plugin's tool from here collides on subrepo names: the
    # plugin's third_party/go and ours both register e.g.
    # third_party/go/github.com_stretchr_testify@windows_amd64, because an arch subrepo's name
    # does not include the subrepo that owns it. That is a bug in Please and not one to fix
    # from inside a packaging script.
    echo "  building $tool for windows_amd64"
    (cd "$path" && "$PLZ" build -p --arch windows_amd64 "//tools/$tool") >/dev/null
    # Named with .exe because Windows will not run a file whose name has no PATHEXT extension.
    # go-rules already names its own output that way; the other two do not, since they pin a
    # released go plugin without that fix.
    built="$path/plz-out/bin/windows_amd64/tools/$tool/$tool"
    [ -f "$built" ] || built="$built.exe"
    [ -f "$built" ] || die "$tool did not build for windows_amd64"
    cp "$built" "$OUT/$tool.exe"
    chmod +w "$OUT/$tool.exe"
  fi

  printf '%-14s %-12s %s  %s\n' "$plugin" "$described" "$sha" "$subject" >> "$OUT/plugin_revisions.txt"
done

echo
echo "vendored into $OUT:"
ls -1 "$OUT"
