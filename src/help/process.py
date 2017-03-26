"""Tool to preprocess some of the help files into an appropriate format."""

import json
import re
import sys


DOCSTRING_RE = re.compile(r'^( *[^ ]+) (\([^\)]+\)):', flags=re.MULTILINE)


def main(filename):
    with open(filename) as f:
        data = json.load(f)
    online_help = lambda f: '${RESET}Online help is available at ${BLUE}https://please.build/lexicon.html#%s${RESET}.\n' % f
    m = lambda k, v: '${BOLD_YELLOW}%s${RESET}(%s)\n\n%s\n\n%s' % (k, ', '.join(
        '${GREEN}%s${RESET}' % a['name'] for a in v['args']), colourise(v['docstring'], v['args'], data['functions']), online_help(k))

    json.dump({
        'topics': {k: m(k, v) for k, v in data['functions'].items()},
        'preamble': '${BOLD_BLUE}%s${RESET} is a built-in build rule in Please. Instructions for use & its arguments:',
    }, sys.stdout, sort_keys=True)


def colourise(docstring, args, functions):
    def replace(m):
        if any(arg for arg in args if arg['name'] == m.group(1).strip() and arg['deprecated']):
            return '${GREY}' + m.group(0)
        return '${YELLOW}%s${RESET} ${GREEN}%s${RESET}:' % (m.group(1), m.group(2))

    # Must order by longest first in case of overlapping options (cgo_library vs. go_library etc)
    for function in sorted(functions, key=lambda f: -len(f)):
        docstring = docstring.replace(function, '${BLUE}%s${RESET}' % function)
    docstring = docstring.replace('Args:\n', '${BOLD_YELLOW}Args:${RESET}\n')
    return DOCSTRING_RE.sub(replace, docstring)


if __name__ == '__main__':
    main(sys.argv[1])
