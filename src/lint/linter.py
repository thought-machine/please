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
UNSORTED_SET_ITERATION = 'unsorted-set-iteration'
UNSORTED_DICT_ITERATION = 'unsorted-dict-iteration'


ERROR_DESCRIPTIONS = {
    SYNTAX_ERROR: 'Syntax error',
    ITERITEMS_USED: 'dict.iteritems called, use dict.items instead',
    ITERVALUES_USED: 'dict.itervalues called, use dict.values instead',
    ITERKEYS_USED: 'dict.iterkeys called, use dict.keys instead (or just iterate the dict)',
    UNSORTED_SET_ITERATION: 'Iteration of sets is not ordered, use sorted()',
    UNSORTED_DICT_ITERATION: 'Iteration of dicts is not ordered, use sorted()',
}

BANNED_ATTRS = {
    'iteritems': ITERITEMS_USED,
    'itervalues': ITERVALUES_USED,
    'iterkeys': ITERKEYS_USED,
}

UNSORTED_CALLS = {
    'set': UNSORTED_SET_ITERATION,
    'dict': UNSORTED_DICT_ITERATION,
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


def _lint(contents):
    try:
        tree = ast.parse(contents)
    except SyntaxError as err:
        yield err.lineno, SYNTAX_ERROR
        return

    for n in ast.walk(tree):
        # .iteritems and so forth
        if isinstance(n, ast.Call) and hasattr(n.func, 'attr') and n.func.attr in BANNED_ATTRS:
            yield n.lineno, BANNED_ATTRS[n.func.attr]
        # Iteration of non-sorted structures
        if isinstance(n, ast.For):
            if isinstance(n.iter, ast.Call) and isinstance(n.iter.func, ast.Name):
                if n.iter.func.id in UNSORTED_CALLS:
                    yield n.lineno, UNSORTED_CALLS[n.iter.func.id]



def lint(filename, suppress=None):
    """Lint the given file. Yields the error codes found."""
    with open(filename) as f:
        contents = f.read()
        # ast discards comments, but we use those to suppress messages.
        lines = contents.split('\n')
    for lineno, code in _lint(contents):
        if not is_suppressed(code, lines[lineno - 1]):
            yield lineno, code


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
    parser.add_argument('--exit_zero', dest='exit_zero', action='store_true')
    parser.set_defaults(exit_zero=False)
    args = parser.parse_args()
    success = all(print_lint(f, args.suppress) for f in args.files)
    sys.exit(0 if success or args.exit_zero else 1)
