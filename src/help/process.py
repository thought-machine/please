"""Tool to preprocess some of the help files into an appropriate format."""

import json
import re
import sys


DOCSTRING_RE = re.compile(r'^( *[^ ]+) (\([^\)]+\)):', flags=re.MULTILINE)


def main(filename):
    with open(filename) as f:
        data = json.load(f)
    m = lambda k, v: '${BOLD_YELLOW}%s${RESET}(%s)\n\n%s' % (k, ', '.join(
        '${BLUE}%s${RESET}' % a['name'] for a in v['args']), colourise(v['docstring']))
    json.dump({
        'topics': {k: m(k, v) for k, v in data['functions'].items()},
        'preamble': '${BOLD_BLUE}%s${RESET} is a built-in build rule in Please. Instructions for use & its arguments:',
    }, sys.stdout, sort_keys=True)


def colourise(docstring):
    docstring = docstring.replace("Args:\n", "${BOLD_YELLOW}Args:${RESET}\n")
    return DOCSTRING_RE.sub(lambda m: '${YELLOW}%s${RESET} ${BLUE}%s${RESET}:' % (m.group(1), m.group(2)), docstring)


if __name__ == '__main__':
    main(sys.argv[1])
