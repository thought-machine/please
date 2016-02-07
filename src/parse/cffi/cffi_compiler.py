"""Compiler script to build the CFFI modules."""

import sys
from cffi import FFI


def main(defs_file):
    ffi = FFI()
    ffi.set_source('parser_interface', None)
    with open(defs_file) as f:
        ffi.cdef(f.read())
    ffi.compile()


if __name__ == '__main__':
    main(sys.argv[1])
