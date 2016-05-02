#!/usr/bin/python

import sys


def main(header):
    with open(header) as f:
        contents = f.read()
    sys.stdout.write(sys.stdin.read().replace('<!-- HEADER -->', contents))


if __name__ == '__main__':
    main(sys.argv[1])
