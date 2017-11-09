#!/usr/bin/env python3
"""Implements a bunch of linters on the repo's source.

Some specific things are run as plz tests (e.g. lint_builtin_rules_test)
but this covers more nebulous things like linters as part of the CI build.

Should be run from the repo root.
"""

import os
import subprocess
import sys
from concurrent import futures
from itertools import groupby


# Dir names that are always blacklisted.
BLACKLISTED_DIRS = {'plz-out', 'test', 'test_data', 'third_party'}


def find_go_files(root):
    for dirname, dirs, files in os.walk(root):
        # Excludes github.com, gopkg.in, etc.
        dirs[:] = [d for d in dirs if '.' not in d and d not in BLACKLISTED_DIRS]
        yield from [(dirname[2:], file) for file in files
                    if file.endswith('.go') and not file.endswith('.bindata.go')]


def run_linters(files):
    try:
        # There are two cases of "possible misuse of unsafe.Pointer" in the parser.
        # We *think* our usage is legit although of course it would be nice to fix
        # the warnings regardless. For now we disable it to get linting of the rest
        # of the package.
        subprocess.check_call(['go', 'tool', 'vet', '-unsafeptr=false'] + files)
        subprocess.check_call(['golint', '-set_exit_status'] + files)
        return True
    except subprocess.CalledProcessError:
        return False


def main():
    all_go_files = [(d, f) for d, f in find_go_files('.')]
    by_dir = [[os.path.join(d, f[1]) for f in files]
              for d, files in groupby(all_go_files, key=lambda x: x[0])]
    with futures.ThreadPoolExecutor() as executor:
        if not all(executor.map(run_linters, by_dir)):
            sys.exit('Some linters failed')


if __name__ == '__main__':
    main()
