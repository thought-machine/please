"""Tool to preprocess some of the help files into an appropriate format."""

import json
import sys


def main(filename):
    with open(filename) as f:
        data = json.load(f)
    m = lambda k, v: '%s(%s)\n\n%s' % (k, ', '.join(a['name'] for a in v['args']), v['docstring'])
    json.dump({
        'topics': {k: m(k, v) for k, v in data['functions'].items()},
        'preamble': '%s is a built-in build rule in Please. Instructions for use & its arguments:',
    }, sys.stdout, sort_keys=True)


if __name__ == '__main__':
    main(sys.argv[1])
