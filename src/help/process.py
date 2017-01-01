"""Tool to preprocess some of the help files into an appropriate format."""

import json
import sys


def main(filename):
    with open(filename) as f:
        data = json.load(f)
    json.dump({k: v['docstring'] for k, v in data['functions'].items()},
              sys.stdout, sort_keys=True)


if __name__ == '__main__':
    main(sys.argv[1])
