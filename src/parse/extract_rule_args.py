"""Script invoked to extract the list of available arguments to each build rule.

This isn't used by Please itself but is useful for other things, e.g. code completion plugins.
"""

import ast
import json
import os
import re
import sys
import textwrap
from itertools import chain


DOCSTRING_RE = re.compile(' *([^ ]+) \\(([^\)]+)\\):( Deprecated)? *([^\n]+(?:\n {8}[^\n]+)*)',
                          flags=re.MULTILINE)


def read_functions(filenames):
    """Reads the given python files and yields the function arguments from them."""
    for filename in filenames:
        # Infer a language from the filename. Theoretically one should derive this from
        # `requires` stanzas but that'd be stupidly hard to do...
        lang, _, _ = filename.partition('_')
        with open(filename) as f:
            tree = ast.parse(f.read(), f.name)
            for i, node in enumerate(ast.iter_child_nodes(tree)):
                if isinstance(node, ast.FunctionDef) and not node.name.startswith('_'):
                    # The c_*** family of functions call through to the cc_family.
                    # They don't have formal argument lists because most things are delegated.
                    if not node.name.startswith('c_'):
                        ds = ast.get_docstring(node)
                        comment, _, _ = ds.partition('\nArgs:\n')
                        comment = textwrap.dedent('    ' + comment.strip())
                        if node.name.startswith('cc_'):
                            alias = node.name.replace('cc_', 'c_')
                            yield node.name, [alias], lang, ds, comment, arg_checks(node)
                            yield alias, [node.name], lang, ds, comment, arg_checks(node)
                        else:
                            yield node.name, None, lang, ds, comment, arg_checks(node)


def arg_checks(node):
    """Yields a sequence of checks on the given ast function node."""
    docs = {m.group(1): (m.group(2), m.group(3), m.group(4))
            for m in DOCSTRING_RE.finditer(ast.get_docstring(node))}
    min_default = len(node.args.args) - len(node.args.defaults)
    # ast in python 3 looks a bit different.
    arg_name = lambda arg: arg.id if hasattr(arg, 'id') else arg.arg
    for i, arg in enumerate(arg_name(arg) for arg in node.args.args):
        if arg.startswith('_'):  # Private, undocumented arguments.
            continue
        assert arg in docs, 'Missing docstring for argument %s to %s()' % (arg, node.name)
        types, deprecated, comment = docs[arg]
        first, _, rest = comment.partition('\n')
        c = (first + '\n' + textwrap.dedent(rest)) if rest else first
        lines = [x.replace('\n', ' ') for x in c.split('.\n')]
        yield arg, i < min_default, types.split(' | '), bool(deprecated), lines


if __name__ == '__main__':
    json.dump({'functions': {
        func_name: {
            'aliases': aliases,
            'comment': comment,
            'docstring': docstring,
            'language': None if language == 'misc' else language,
            'args': [{
                'name': arg_name,
                'required': required,
                'types': types,
                'deprecated': deprecated,
                'comment': comment,
            } for arg_name, required, types, deprecated, comment in func_info],
        } for func_name, aliases, language, docstring, comment, func_info in read_functions(sys.argv[1:])
    }}, sys.stdout, sort_keys=True, indent=4)
