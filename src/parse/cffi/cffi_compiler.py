"""Compiler script to build the CFFI modules."""

import os
import platform
import sys
from cffi import FFI

def main(defs_file, parser_file, verbose):
    ffi = FFI()
    with open(defs_file) as f:
        ffi.embedding_api(f.read())
    ldflags = None
    # Support downloading portable PyPy on Linux.
    if platform.python_implementation() == 'PyPy' and platform.system() == 'Linux':
        ldflags = ['-Wl,-rpath=./plz-out/gen/_remote/_pypy/bin']
    if platform.python_implementation() == 'CPython' and platform.system() == 'Darwin':
        version = platform.python_version_tuple()
        ldflags = ['-L' + os.path.join(sys.prefix, 'lib')]
    ffi.set_source('parser_interface', '#include "%s"' % defs_file,
                   extra_link_args=ldflags)
    with open(parser_file) as f:
        ffi.embedding_init_code(f.read())
    interpreter, _, _ = os.path.basename(sys.executable).partition('.')
    ffi.compile(target='libplease_parser_%s.*' % interpreter, verbose=verbose)


if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2], len(sys.argv) > 3 and sys.argv[3] == '--verbose')
