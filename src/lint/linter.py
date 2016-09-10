#!/usr/bin/python
"""A linter for files written in Please's BUILD language.

Most features of the language we attempt to control at parse time, for example
banning import and print statements, but some cannot be readily or reliably
identified then (e.g. use of dict.iteritems in a python 2 based interpreter).

This script attempts to identify such stylistic things as a linter. The
current things searched for are:
 - Use of dict.iteritems, dict.itervalues and dict.iterkeys; you should
   prefer .items, .values and .keys respectively.
   These are conceptually not supported and would be removed from the
   BUILD language, but that has proven technically difficult.
"""

import argparse
import ast
import re
import sys


SYNTAX_ERROR = 'syntax-error'
ITERITEMS_USED = 'iteritems-used'
ITERVALUES_USED = 'itervalues-used'
ITERKEYS_USED = 'iterkeys-used'


ERROR_DESCRIPTIONS = {
    SYNTAX_ERROR: 'Syntax error',
    ITERITEMS_USED: 'dict.iteritems called, use dict.items instead',
    ITERVALUES_USED: 'dict.itervalues called, use dict.values instead',
    ITERKEYS_USED: 'dict.iterkeys called, use dict.keys instead (or just iterate the dict)',
}

BANNED_ATTRS = {
    'iteritems': ITERITEMS_USED,
    'itervalues': ITERVALUES_USED,
    'iterkeys': ITERKEYS_USED,
}


def walk(n):
    for node in ast.iter_child_nodes(n):
        print('%s %s' % (getattr(node, 'lineno', '?'), node))
        if isinstance(node, (ast.Attribute, ast.Call)):
            print dir(node)
            print node.value
        walk(node)


def is_suppressed(code, line):
    """Returns True if the given code is suppressed on this line."""
    if '#' not in line:
        return False
    comment = line[line.index('#') + 1:]
    return 'nolint' in comment or re.search('lint: *disable=' + code, comment)


def lint(filename, suppress=None):
    """Lint the given file. Yields the error codes found."""
    with open(filename) as f:
        contents = f.read()
        # ast discards comments, but we use those to suppress messages.
        lines = contents.split('\n')
        try:
            tree = ast.parse(contents, filename)
        except SyntaxError as err:
            yield err.lineno, SYNTAX_ERROR
            return

    for node in ast.walk(tree):
        if isinstance(node, ast.Call) and node.func.attr in BANNED_ATTRS:
            if not is_suppressed(BANNED_ATTRS[node.func.attr], lines[node.lineno - 1]):
                yield node.lineno, BANNED_ATTRS[node.func.attr]



def print_lint(filename, suppress=None):
    """Lint the given file and print results. Returns True if no errors were found."""
    success = True
    for lineno, code in lint(filename, suppress):
        sys.stdout.write('L%d:%s: %s\n' % (lineno, code, ERROR_DESCRIPTIONS[code]))
        success = False
    return success


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('files', nargs='+')
    parser.add_argument('--suppress', nargs='+')
    parser.add_argument('--exit_code', dest='exit_code', action='store_true')
    parser.set_defaults(exit_code=False)
    args = parser.parse_args()
    success = all(lint(f, args.suppress) for f in args.files)
    sys.exit(0 if success or not exit_code else 1)
