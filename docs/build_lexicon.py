#!/usr/bin/python
"""Script to autogenerate an initial version of the lexicon from docstrings.

This pretty much shows my essential laziness in that I couldn't be bothered
writing all this up in HTML by hand.
"""

import ast
import os
import re
import sys


def read_all_functions(path):
    """Yields all functions contained in the builtin rule set."""
    for filename in os.listdir(path):
        if filename.endswith('.py'):
            with open(os.path.join(path, filename)) as f:
                sys.stdout.write('\n    <h2><a name="%s">%s</a></h2>\n' % (
                    filename[:filename.find('_')],
                    filename[:-3].replace('_', ' ').capitalize()
                ))
                tree = ast.parse(f.read(), f.name)
                for node in ast.iter_child_nodes(tree):
                    if not isinstance(node, ast.FunctionDef):
                        continue
                    # Exclude anything ffi related, it'll be a callback
                    if 'ffi' in (d.func.value.id for d in node.decorator_list):
                        continue
                    if node.name.startswith('_'):
                        continue
                    # Pad for args that don't have a default.
                    padding = len(node.args.args) - len(node.args.defaults)
                    args = [(arg.id, _get_default(default)) for arg, default in
                            zip(node.args.args, [''] * padding + node.args.defaults)]
                    yield node.name, args, ast.get_docstring(node)


def _get_default(default):
    """Translates an AST node to a string for a default function argument."""
    if isinstance(default, str):
        return default
    for guess in ['id', 'n', 'value', 's']:
        if hasattr(default, guess):
            ret = getattr(default, guess)
            if 'ast.' in str(ret):
                return 'Set in config'
            return ret
    raise ValueError('Don\'t know how to translate %s' % default)


def to_html(name, args, docstring):
    """Produces HTML version of a function."""
    descs = dict(parse_docstring(name, docstring))
    data = {
        'function_name': name,
        'overview': descs.get('Overview'),
        'arglist': ', '.join('%s=%s' % (arg, default) if default else arg
                             for arg, default in args),
        'rows': ''.join(ROW_TEMPLATE % (arg,
                                        default or '',
                                        descs.get(arg, [''])[0],
                                        descs.get(arg, ['', ''])[1])
                        for arg, default in args),
    }
    sys.stdout.write(TEMPLATE % data)


def parse_docstring(name, docstring):
    """Parses a Google style docstring into a sequence of function arguments.

    The overview is named Overview, and there's no effort made to handle Returns, Raises
    or Yields sections, or other odd formatting - ie. it's only as robust as it needs to be.

    TODO(pebers): gonna write a grm for this one day, don't like this at all.
    """
    if not docstring:
        sys.stderr.write('Missing docstring for %s\n' % name)
        yield ('Overview', '')
    elif 'Args:' not in docstring:
        sys.stderr.write('Missing args section in docstring for %s\n' % name)
        yield ('Overview', docstring)
    else:
        before, sep, after = docstring.partition('Args:')
        yield ('Overview', before.strip())
        s = []
        name = ''
        arg_type = ''
        for line in after.split('\n'):
            match = re.match(r' *([^:\(]+) \(([^\)]+)\): *(.*)', line)
            if match:
                if s:
                    yield name, (arg_type, '\n'.join(s))
                    s = []
                name = match.group(1)
                arg_type = match.group(2)
                s.append(match.group(3))
            else:
                s.append(line)
        yield name, (arg_type, '\n'.join(s))


TEMPLATE = """
    <h3>%(function_name)s</h3>

    <p><code>%(function_name)s(%(arglist)s)</code></p>

    <p>%(overview)s</p>

    <table>
      <tr>
	<th>Argument</th>
	<th>Default</th>
	<th>Type</th>
	<th></th>
      </tr>
      %(rows)s
    </table>
"""

ROW_TEMPLATE = """
      <tr>
	<td>%s</td>
	<td>%s</td>
	<td>%s</td>
	<td>%s</td>
      </tr>
"""


if __name__ == '__main__':
    for function in read_all_functions('src/parse/rules'):
        to_html(*function)
