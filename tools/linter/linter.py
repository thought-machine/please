#!/usr/bin/python
"""A linter for files written in Please's BUILD language.

Most features of the language we attempt to control at parse time, for example
banning import and print statements, but some cannot be readily or reliably
identified then (e.g. use of dict.iteritems in a python 2 based interpreter).

The linter attempts to identify such stylistic things. The current things searched for are:
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
   Similarly we check for arguments that are deprecated.
 - Detection of incorrect arguments to builtin functions. Note that we don't bother
   doing typechecking on them - that's done at runtime anyway more accurately than
   we can here. This is somewhat unnecessary but again may help some workflows
   where the linter might hint you not to commit with obvious problems.
 - Missing required arguments to builtin functions. This will of course fail at
   parse time but it still seems useful to collect here.
 - Detection of duplicates in argument lists - most usually this is useful for deps,
   but it applies to anything since there's no reason to have a duplicate in any
   argument list to any builtin Please function.
 - Some specific checks on third-party library rules (maven_jar, pip_library and
   go_get) that warn on duplicated artifacts.
   This is deliberately version-agnostic, since in most cases having two different
   versions of the same library turns out to be a Bad Thing, although there are
   potentially legitimate uses for that as well. In such cases we suggest splitting
   into different packages if the linter is becoming vexing.

Lint warnings can be suppressed on a per-line basis by adding a trailing comment
saying either `# nolint` or `# lint:disable=iterkeys-used`, or on a per-file basis
by adding lines containing only the disabling messages. Currently you can't use
`nolint` on a per-file basis; consider not running the linter on that file... :)

Usage:

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
DEPRECATED_ARGUMENT = 'deprecated-argument'
MISSING_ARGUMENT = 'missing-argument'
INCORRECT_ARGUMENT = 'incorrect-argument'
UNNECESSARY_DUPLICATE = 'unnecessary-duplicate'
DUPLICATE_ARTIFACT = 'duplicate-artifact'


ERROR_DESCRIPTIONS = {
    SYNTAX_ERROR: 'Syntax error',
    ITERITEMS_USED: 'dict.iteritems called, use dict.items instead',
    ITERVALUES_USED: 'dict.itervalues called, use dict.values instead',
    ITERKEYS_USED: 'dict.iterkeys called, use dict.keys instead (or just iterate the dict)',
    UNSORTED_SET_ITERATION: 'Iteration of sets is not ordered, use sorted()',
    UNSORTED_DICT_ITERATION: 'Iteration of dicts is not ordered, use sorted()',
    NON_KEYWORD_CALL: 'Call to builtin rule without using keyword arguments',
    DEPRECATED_FUNCTION: 'The function include_defs is deprecated, use subinclude() instead',
    DEPRECATED_ARGUMENT: 'Deprecated argument',
    MISSING_ARGUMENT: 'Missing required argument',
    INCORRECT_ARGUMENT: 'Unknown argument to built-in function',
    UNNECESSARY_DUPLICATE: 'Unnecessary duplicate in argument',
    DUPLICATE_ARTIFACT: 'Duplicated third-party artifact',
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


def _args(func):
    """Alters argument structure on function object to be a map of name -> arg instead of a list."""
    func['args'] = {arg['name']: arg for arg in func['args']}
    return func


def _extract_keyword(node, arg, default=''):
    """Extracts a single keyword argument from an AST node."""
    for kwd in node.keywords:
        if kwd.arg == arg and hasattr(kwd.value, 's'):
            return kwd.value.s
    return default


JSON = json.loads(pkg_resources.resource_string('src.parse', 'rule_args.json').decode('utf-8'))
WHITELISTED_FUNCTIONS = {'subinclude', 'glob', 'include_defs', 'licenses'}
BUILTIN_FUNCTIONS = {k: _args(v) for k, v in JSON['functions'].items() if k not in WHITELISTED_FUNCTIONS}
DEPRECATED_FUNCTIONS = {'include_defs'}
THIRD_PARTY_FUNCTIONS = {
    'maven_jar': lambda n: _extract_keyword(n, 'id').rpartition(':')[0],
    'go_get': lambda n: _extract_keyword(n, 'get'),
    'pip_library': lambda n: _extract_keyword(n, 'package_name', _extract_keyword(n, 'name')),
    'python_wheel': lambda n: _extract_keyword(n, 'package_name', _extract_keyword(n, 'name')),
}
third_party_artifacts = set()


def is_suppressed(code, line, file_suppressions):
    """Returns True if the given code is suppressed on this line."""
    if code in file_suppressions:
        return True
    if '#' not in line:
        return False
    comment = line[line.index('#') + 1:]
    return 'nolint' in comment or re.search('lint: *disable=' + code, comment)


def _lint_iteritems(n):
    """Lints for calls to dict.iteritems and so forth."""
    if isinstance(n, ast.Call) and hasattr(n.func, 'attr') and n.func.attr in BANNED_ATTRS:
        yield n.lineno, BANNED_ATTRS[n.func.attr]


def _lint_unsorted_iteration(n):
    """Lints for iteration of unsorted structures."""
    if isinstance(n, ast.For):
        if isinstance(n.iter, ast.Call) and isinstance(n.iter.func, ast.Name):
            if n.iter.func.id in UNSORTED_CALLS:
                yield n.lineno, UNSORTED_CALLS[n.iter.func.id]
        elif isinstance(n.iter, (ast.Dict, ast.DictComp)):
            yield n.lineno, UNSORTED_DICT_ITERATION
        elif isinstance(n.iter, (ast.Set, ast.SetComp)):
            yield n.lineno, UNSORTED_SET_ITERATION


def _lint_builtin_functions(n):
    """Lints for incorrect calls to builtin functions."""
    if isinstance(n, ast.Call) and isinstance(n.func, ast.Name) and n.func.id in BUILTIN_FUNCTIONS:
        if n.args or n.starargs:
            yield n.lineno, NON_KEYWORD_CALL
        args = BUILTIN_FUNCTIONS[n.func.id]['args']
        for kwd in n.keywords or []:
            if kwd.arg not in args and not kwd.arg.startswith('_'):
                yield kwd.value.lineno, INCORRECT_ARGUMENT
        # Don't check kwargs if the caller is doing a **kwargs into it, assume that'll take care of it.
        if not n.kwargs:
            kwds = {kwd.arg for kwd in n.keywords or []}
            for name, arg in args.items():
                if arg['required'] and name not in kwds:
                    yield n.lineno, MISSING_ARGUMENT


def _lint_deprecated_functions(n):
    """Lints for calls to deprecated functions."""
    if isinstance(n, ast.Call) and isinstance(n.func, ast.Name) and n.func.id in DEPRECATED_FUNCTIONS:
        yield n.lineno, DEPRECATED_FUNCTION


def _lint_deprecated_args(n):
    """Lints for calls to builtin functions that use deprecated arguments."""
    if isinstance(n, ast.Call) and isinstance(n.func, ast.Name) and n.func.id in BUILTIN_FUNCTIONS:
        args = BUILTIN_FUNCTIONS[n.func.id]['args']
        for kwd in n.keywords or []:
            if args.get(kwd.arg, {}).get('deprecated'):
                yield kwd.value.lineno, DEPRECATED_ARGUMENT


def _lint_duplicates(n):
    """Lints for duplicates in deps, srcs etc."""
    if isinstance(n, ast.Call):
        for kwd in n.keywords or []:
            if isinstance(kwd.value, ast.List):
                seen = set()
                for s in kwd.value.elts:
                    if isinstance(s, ast.Str):
                        if s.s in seen:
                            yield s.lineno, UNNECESSARY_DUPLICATE
                        seen.add(s.s)


def _lint_third_party_artifacts(n):
    """Lints for duplicate third-party artifacts (in pip_library, maven_jar etc)."""
    if isinstance(n, ast.Call) and isinstance(n.func, ast.Name) and n.func.id in THIRD_PARTY_FUNCTIONS:
        artifact = THIRD_PARTY_FUNCTIONS[n.func.id](n)
        if artifact:
            if artifact in third_party_artifacts:
                yield n.lineno, DUPLICATE_ARTIFACT
            else:
                third_party_artifacts.add(artifact)


def _lint(contents):
    try:
        tree = ast.parse(contents)
    except SyntaxError as err:
        yield err.lineno, SYNTAX_ERROR
        return
    for n in ast.walk(tree):
        for fn in LINT_FUNCTIONS:
            for error in fn(n):
                yield error


LINT_FUNCTIONS = [
    _lint_iteritems, _lint_unsorted_iteration, _lint_builtin_functions,
    _lint_deprecated_functions, _lint_deprecated_args, _lint_duplicates,
    _lint_third_party_artifacts,
]


def lint(filename, suppress=None):
    """Lint the given file. Yields the error codes found."""
    third_party_artifacts.clear()  # Only check this within a single file.
    with open(filename) as f:
        contents = f.read()
    # ast discards comments, but we use those to suppress messages.
    lines = contents.split('\n')
    # Find any lines that fully suppress messages.
    matches = [re.match('^ *# lint: *disable=(.*)$', line) for line in lines]
    suppressions = {match.group(1) for match in matches if match}.union(suppress or [])
    for lineno, code in _lint(contents):
        if not is_suppressed(code, lines[lineno - 1], suppressions):
            yield lineno, code


def print_lint(filenames, suppress=None):
    """Lint the given files and print results. Returns True on success."""
    success = True
    for filename in filenames:
        for lineno, code in lint(filename, suppress):
            sys.stdout.write('%s,L%d: %s: %s\n' % (filename, lineno, code, ERROR_DESCRIPTIONS[code]))
            success = False
    return success


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Linter for Please BUILD files')
    parser.add_argument('files', nargs='+')
    parser.add_argument('--suppress', nargs='+')
    parser.add_argument('--exit_zero', dest='exit_zero', action='store_true')
    parser.set_defaults(exit_zero=False)
    args = parser.parse_args()
    success = print_lint(args.files, args.suppress)
    sys.exit(0 if success or args.exit_zero else 1)
