"""Script invoked at build time to add type assertions to builtin rules.

PEP3107 / PEP0484 style hints would be nicer, of course, but we need to
retain python 2 compatibility for the foreseeable future.
We could also write them manually, but I've chosen not to because
of laziness.
"""

import ast
import os
import re
import sys


DOCSTRING_RE = re.compile(r' *([^ ]+) \(([^\)]+)\):')


def read_functions(filename):
    """Reads the given python file and yields the function arguments in it."""
    with open(filename) as f:
        tree = ast.parse(f.read(), f.name)
        for i, node in enumerate(ast.iter_child_nodes(tree)):
            if isinstance(node, ast.FunctionDef) and not node.name.startswith('_'):
                yield node.body[0].value.lineno, list(arg_checks(node))


def arg_checks(node):
    """Yields a sequence of checks on the given ast function node."""
    docs = {m.group(1): m.group(2) for m in DOCSTRING_RE.finditer(ast.get_docstring(node))}
    min_default = len(node.args.args) - len(node.args.defaults)
    # ast in python 3 looks a bit different.
    arg_name = lambda arg: arg.id if hasattr(arg, 'id') else arg.arg
    for i, arg in enumerate(arg_name(arg) for arg in node.args.args):
        assert arg in docs, 'Missing docstring for argument %s to %s()' % (arg, node.name)
        doc = docs[arg]
        rtype = doc.replace('bool', 'int')  # Bools are ints so an int is acceptable.
        if '|' in doc:
            types = rtype.split(' | ')
            yield 'assert not %s or isinstance(%s, (%s)), "Argument %s to %s must be a %s"' % (
                arg, arg, ', '.join(types), arg, node.name, doc.replace('|', 'or'))
        elif i >= min_default:
            if doc == 'function':
                # Have to check functions a bit specially. Maybe we should document them
                # as 'callable' instead of 'function'?
                yield 'assert not %s or callable(%s), "Argument %s to %s must be callable"' % (
                    arg, arg, arg, node.name)
            else:
                yield 'assert not %s or isinstance(%s, %s), "Argument %s to %s must be a %s"' % (
                    arg, arg, rtype, arg, node.name, doc)
        else:
            yield 'assert isinstance(%s, %s), "Argument %s to %s must be a %s"' % (
                arg, rtype, arg, node.name, doc)


def process(filename):
    checks = dict(read_functions(filename))
    with open(filename) as f, open(os.path.basename(filename), 'w') as f2:
        for i, line in enumerate(f):
            if i in checks:
                f2.write('\n'.join(indent * ' ' + check for check in checks[i]) + '\n\n')
            f2.write(line)
            indent = len(line) - len(line.lstrip())


if __name__ == '__main__':
    for filename in sys.argv[1:]:
        process(filename)
