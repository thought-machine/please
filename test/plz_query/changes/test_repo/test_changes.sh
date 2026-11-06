#!/usr/bin/env bash
# Checks that `plz query changes --before_hashes` gives the same results as parsing the base revision.

set -euo pipefail

# The repo's files are hardlinked to the originals, so break the links before modifying anything.
find . -type f -exec sh -c 'cp -p "$1" "$1.tmp" && mv "$1.tmp" "$1"' _ {} \;

export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com

# Snapshots are deliberately written into the repo root without being ignored, as CI would do when it
# downloads them; they shouldn't count towards the working tree being dirty.
# Command output goes in out/, which is ignored, since the shell creates it before plz can see it.
mkdir out
git init -q
git symbolic-ref HEAD refs/heads/master
printf 'plz-out\nout\n' > .gitignore
git add -A
git commit -qm base
$TOOLS_PLEASE query hashes --output_file base.snapshot

git checkout -q -b feature
echo changed > a/a.txt
sed -i.bak 's/echo c/echo C/' b/BUILD_FILE && rm b/BUILD_FILE.bak
echo 'genrule(name="d", outs=["d.out"], cmd="echo d > $OUT")' >> b/BUILD_FILE
git commit -qam feature

check() {
    local expected="$1"
    local actual="$2"
    if [ "$(cat "$actual")" != "$expected" ]; then
        printf 'Unexpected output in %s; expected:\n%s\n---- got ----\n' "$actual" "$expected" >&2
        cat "$actual" >&2
        exit 1
    fi
}

# --strict_hashes fails if the snapshot is rejected, so success here means it was actually used.
for level in 0 1 -1; do
    $TOOLS_PLEASE query changes --since master --level $level > out/without_$level.txt
    $TOOLS_PLEASE query changes --since master --level $level --before_hashes base.snapshot --strict_hashes > out/with_$level.txt
    diff -u out/without_$level.txt out/with_$level.txt
done
check $'//a:a\n//b:c\n//b:d' out/with_0.txt
check $'//a:a\n//b:b\n//b:c\n//b:d' out/with_-1.txt

# A snapshot written by query changes is itself usable as a base.
$TOOLS_PLEASE query changes --since master --before_hashes base.snapshot --strict_hashes --write_after_hashes after.snapshot > /dev/null
git checkout -q -b feature2
sed -i.bak 's/echo d/echo D/' b/BUILD_FILE && rm b/BUILD_FILE.bak
git commit -qam feature2
$TOOLS_PLEASE query changes --since feature --before_hashes after.snapshot --strict_hashes > out/feature2.txt
# after.snapshot is an untracked input of //:snap, but it was named on the command line so isn't a changed file.
check '//b:d' out/feature2.txt

# after.snapshot wasn't generated at the merge base with master, so it must be rejected...
if $TOOLS_PLEASE query changes --since master --before_hashes after.snapshot --strict_hashes > /dev/null 2>&1; then
    echo "Expected snapshot at the wrong revision to be rejected with --strict_hashes" >&2
    exit 1
fi
# ...and without --strict_hashes we fall back to parsing, and get the same answer as not using it at all.
# (Without naming it, after.snapshot is an ordinary untracked file so //:snap is reported as changed.)
$TOOLS_PLEASE query changes --since master --level -1 | grep -vx '//:snap' > out/fallback_expected.txt
$TOOLS_PLEASE query changes --since master --level -1 --before_hashes after.snapshot > out/fallback.txt
diff -u out/fallback_expected.txt out/fallback.txt
[ "$(git branch --show-current)" = feature2 ] || { echo "Didn't end up back on the original branch" >&2; exit 1; }
