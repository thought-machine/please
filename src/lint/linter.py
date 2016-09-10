#!/usr/bin/python
"""A linter for files written in Please's BUILD language.

Most features of the language we attempt to control at parse time, for example
banning import and print statements, but some cannot be readily or reliably
identified then (e.g. use of dict.iteritems in a python 2 based interpreter).

This script attempts to identify such stylistic things as a linter. The
current things searched for are:
 - Syntax errors - obviously these are caught at build time too, but we have
   to handle them here, and it's useful in some workflows to have lint prompt
   for this as well.
 - Use of dict.iteritems, dict.itervalues and dict.iterkeys; you should
   prefer .items, .values and .keys respectively.
   These are conceptually not supported and would be removed from the
   BUILD language, but that has proven technically difficult.
 - Iteration of sets and dicts without using sorted() - their order is
   undetermined, and while we attempt to keep it consistent between runs,
   it might not be the same between implementations. In a system where
   determinism is important you're better off using sorted().
   This check isn't 100% reliable, it's very complex to catch all possible cases.
 - Calling builtin build rules through non-keyword arguments.
   We don't specifically guarantee argument order and suggest that they should
   always be called with keyword arguments, but it is possible to call them without.
 - Deprecation warnings for functions that are retained but no longer recommended.
   The most obvious case here is include_defs which we haven't removed because it's
   used internally for some Bazel compatibility, and it might be useful for Buck
   compatibility.
"""

import argparse
import ast
import json
import pkg_resources
import re
import sys


SYNTAX_ERROR = 'syntax-error'
ITERITEMS_USED = 'iteritems-used'
ITERVALUES_USED = 'itervalues-used'
ITERKEYS_USED = 'iterkeys-used'
UNSORTED_SET_ITERATION = 'unsorted-set-iteration'
UNSORTED_DICT_ITERATION = 'unsorted-dict-iteration'
NON_KEYWORD_CALL = 'non-keyword-call'
DEPRECATED_FUNCTION = 'deprecated-function'


ERROR_DESCRIPTIONS = {
    SYNTAX_ERROR: 'Syntax error',
    ITERITEMS_USED: 'dict.iteritems called, use dict.items instead',
    ITERVALUES_USED: 'dict.itervalues called, use dict.values instead',
    ITERKEYS_USED: 'dict.iterkeys called, use dict.keys instead (or just iterate the dict)',
    UNSORTED_SET_ITERATION: 'Iteration of sets is not ordered, use sorted()',
    UNSORTED_DICT_ITERATION: 'Iteration of dicts is not ordered, use sorted()',
    NON_KEYWORD_CALL: 'Call to builtin rule without using keyword arguments',
    DEPRECATED_FUNCTION: 'The function include_defs is deprecated, use subinclude() instead'
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

JSON = json.loads(pkg_resources.resource_string('src.parse', 'rule_args.json'))
WHITELISTED_FUNCTIONS = {'subinclude', 'glob', 'include_defs', 'licenses'}
BUILTIN_FUNCTIONS = {k: v for k, v in JSON['functions'].items() if k not in WHITELISTED_FUNCTIONS}
DEPRECATED_FUNCTIONS = {'include_defs'}


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
            elif isinstance(n.iter, (ast.Dict, ast.DictComp)):
                yield n.lineno, UNSORTED_DICT_ITERATION
            elif isinstance(n.iter, (ast.Set, ast.SetComp)):
                yield n.lineno, UNSORTED_SET_ITERATION
        # Builtin argument calls
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Name) and n.func.id in BUILTIN_FUNCTIONS:
            if n.args or n.starargs:
                yield n.lineno, NON_KEYWORD_CALL
        # Deprecated functions
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Name) and n.func.id in DEPRECATED_FUNCTIONS:
            yield n.lineno, DEPRECATED_FUNCTION


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
        sys.stdout.write('%s,L%d: %s: %s\n' % (filename, lineno, code, ERROR_DESCRIPTIONS[code]))
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
