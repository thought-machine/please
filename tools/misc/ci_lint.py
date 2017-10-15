#!/usr/bin/python3
"""Implements a bunch of linters on the repo's source.

Some specific things are run as plz tests (e.g. lint_builtin_rules_test)
but this covers more nebulous things like linters as part of the CI build.

Should be run from the repo root.
"""

import os
import subprocess
import sys
from itertools import groupby


# Packages that don't load. Will be cleaned up over time.
BLACKLISTED_VET_PACKAGES = {
    'src/cache/server',
    'tools/please_diff_graphs',
    'tools/please_pex',
    'tools/please_maven',
    'tools/please_go_test',
    'tools/please_go_test/test_data',
    'test/go_rules',
    'test/go_rules/test',
    'third_party/go/zip',
    # There are two warnings in here about unsafe.Pointer; we *think* they
    # are safe but not 100% sure and it'd be nice to clean it up.
    # Unfortunately may be hard due to cgo checks :(
    'src/parse',
}


def find_go_files(root):
    for dirname, dirs, files in os.walk(root):
        # Excludes github.com, gopkg.in, etc.
        dirs[:] = [d for d in dirs if '.' not in d and d != 'plz-out']
        yield from [(dirname[2:], file) for file in files if file.endswith('.go')]


def run_linters(files):
    try:
        subprocess.check_call(['go', 'tool', 'vet'] + files)
        subprocess.check_call(['golint', '-set_exit_status'] + files)
        return True
    except subprocess.CalledProcessError:
        return False


def main():
    all_go_files = [(d, f) for d, f in find_go_files('.') if d not in BLACKLISTED_VET_PACKAGES]
    by_dir = [[os.path.join(d, f[1]) for f in files]
              for d, files in groupby(all_go_files, key=lambda x: x[0])]
    if not all([run_linters(files) for files in by_dir]):
        sys.exit('Some linters failed')


if __name__ == '__main__':
    main()
