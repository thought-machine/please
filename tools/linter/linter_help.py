"""Trivial script to extract the usage info from the linter and write it in our help format."""

import ast
import json
import sys


if __name__ == '__main__':
    doc = ast.get_docstring(ast.parse(sys.stdin.read()))
    json.dump({'preamble': '%s refers to the Please BUILD file linter', 'topics': {
        name: doc for name in ['lint', 'linter', 'plz_build_lint']
    }}, sys.stdout)
