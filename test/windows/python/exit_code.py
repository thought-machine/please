"""Exits with the code it is given, so that something can check it arrives."""

import sys

if __name__ == '__main__':
    sys.exit(int(sys.argv[1]))
