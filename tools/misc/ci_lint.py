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
# Count of packages linted so far and total to do
done = 0
total = 0


class Linter:

    def __init__(self, root):
        all_go_files = [(d, f) for d, f in self.find_go_files(root)]
        self.by_dir = [[os.path.join(d, f[1]) for f in files]
                       for d, files in groupby(all_go_files, key=lambda x: x[0])]
        self.done = 0
        self.write('Linting... 0 / %d done', len(self.by_dir))

    def write(self, msg, args, overwrite=False):
        if sys.stderr.isatty() and not os.environ.get('CI'):
            if overwrite:
                sys.stderr.write('\033[1G')
            sys.stderr.write(msg % args)
            sys.stderr.flush()

    def reset_line(self):
        if sys.stderr.isatty() and not os.environ.get('CI'):
            sys.stderr.write('\033[1G')

    def find_go_files(self, root):
        for dirname, dirs, files in os.walk(root):
            # Excludes github.com, gopkg.in, etc.
            dirs[:] = [d for d in dirs if '.' not in d and d not in BLACKLISTED_DIRS]
            yield from [(dirname[2:], file) for file in files
                        if file.endswith('.go') and not file.endswith('.bindata.go')]

    def run_linters(self, files):
        try:
            # There are two cases of "possible misuse of unsafe.Pointer" in the parser.
            # We *think* our usage is legit although of course it would be nice to fix
            # the warnings regardless. For now we disable it to get linting of the rest
            # of the package.
            subprocess.check_call(['go', 'tool', 'vet', '-unsafeptr=false', '-structtags=false'] + files)
            subprocess.check_call(['golint', '-set_exit_status'] + files)
            return True
        except subprocess.CalledProcessError:
            return False
        finally:
            self.done += 1
            self.write('Linting... %d / %d done', (self.done, len(self.by_dir)), overwrite=True)


def main():
    linter = Linter('.')
    with futures.ThreadPoolExecutor(max_workers=6) as executor:
        if not all(executor.map(linter.run_linters, linter.by_dir)):
            linter.reset_line()
            sys.exit('Some linters failed')
        linter.reset_line()
    # Run Buildifier
    try:
        subprocess.check_call(['plz-out/bin/src/please', 'run', '//tools/misc:buildify', '-p', '--', '--mode=check'],
                              stdout=sys.stdout, stderr=sys.stderr)
    except subprocess.CalledProcessError:
        sys.exit('Your BUILD files are not correctly formatted, run `plz buildify` to fix.')


if __name__ == '__main__':
    main()
