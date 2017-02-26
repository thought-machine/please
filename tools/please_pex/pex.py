#!/usr/bin/env python
"""Pex building script for Please."""

import argparse
import os
import shutil
import sys
import zipfile

# We need to load pex; it must be shipped in .bootstrap so it works when it loads,
# but that's already been scrubbed from the path so we must re-add it here.
sys.path.insert(0, os.path.abspath(os.path.join(sys.argv[0], '.bootstrap')))

from _pex.pex_builder import PEXBuilder
from _pex.interpreter import PythonInterpreter

# Global reference to our zipfile, which we extract various bits from.
please_pex_zipfile = None

USAGE = """please_pex is a tool for Please to build pex files.

Due to some optimisations in how we compile pex files, this is now only used
to build the entry point and bootstrap code. Other python_library rules are
concatenated together with it at the last minute to produce the final pex file.

Typically this is not invoked on its own, Please will do it for you.
"""


def dereference_symlinks(src):
    """
    Resolve all symbolic references that `src` points to.  Note that this
    is different than `os.path.realpath` as path components leading up to
    the final location may still be symbolic links.
    """
    while os.path.islink(src):
        src = os.path.join(os.path.dirname(src), os.readlink(src))
    return src


def add_directory(root_path, path, pex_builder, exclude=None):
    """Recursively adds the contents of a directory to the pex."""
    base = os.path.join(root_path, path)
    for filename in os.listdir(base):
        if filename == exclude:
            continue
        src = dereference_symlinks(os.path.join(base, filename))
        dst = os.path.join(path, filename)
        if os.path.isdir(src):
            add_directory(root_path, dst, pex_builder)
        elif filename.endswith('.py'):
            # Assume this is source code and add as such.
            # TODO(pebers): this is possibly a bit simplistic...
            pex_builder.add_source(src, dst)
        elif not filename.endswith('.pyc'):
            pex_builder.add_resource(src, dst)


def extract_recursive_dir(dirname, out_dir):
    """Extracts an entire directory and copies it to the output dir."""
    for info in please_pex_zipfile.infolist():
        if info.filename.startswith(dirname) and not info.filename.endswith('.pyc'):
            please_pex_zipfile.extract(info, path=out_dir)


def extract_file(filename, out_dir, out_file=None, replacements=None):
    """Extracts a single file from the pex and writes it into the out dir."""
    if not replacements:
        please_pex_zipfile.extract(filename, path=out_dir)
        return
    # More elaborate version if we need to replace the contents of the file.
    contents = please_pex_zipfile.read(filename)
    for k, v in replacements.items():
        contents = contents.replace(k.encode('utf-8'), v.encode('utf-8'))
    with open(os.path.join(out_dir, out_file or filename), 'wb') as f:
        f.write(contents)


def remove_ext(x):
    """Strip a .py file extension if present."""
    return x[:-3] if x.endswith('.py') else x


def main(args):
    global please_pex_zipfile
    if not args.bootstrap and not please_pex_zipfile:
        please_pex_zipfile = zipfile.ZipFile(sys.argv[0])

    # Pex doesn't support relative interpreter paths.
    if not args.interpreter.startswith('/'):
        try:
            args.interpreter = shutil.which(args.interpreter)
        except AttributeError:
            # not python3.
            from distutils import spawn
            args.interpreter = spawn.find_executable(args.interpreter)

    if not args.bootstrap:
        # Add pkg_resources, the bootstrapper and the required main files.
        # If we're creating a test we can just extract the entire bootstrap directory.
        if args.test_srcs:
            extract_recursive_dir('.bootstrap', args.src_dir)
            test_modules = ','.join(remove_ext(src.replace('/', '.'))
                                    for src in args.test_srcs.split(','))
            extract_file('tools/please_pex/test_main.py', args.src_dir, 'test_main.py', {
                '__TEST_NAMES__': test_modules,
            })
        else:
            extract_recursive_dir('.bootstrap/_pex', args.src_dir)
            extract_recursive_dir('.bootstrap/pkg_resources', args.src_dir)
            extract_file('.bootstrap/six.py', args.src_dir)
        extract_file('tools/please_pex/pex_main.py', args.src_dir, 'pex_main.py', {
            '__MODULE_DIR__': args.module_dir,
            '__ENTRY_POINT__': remove_ext(args.entry_point.replace('/', '.')),
            '__ZIP_SAFE__': str(args.zip_safe),
        })

    # Setup a temp dir that the PEX builder will use as its scratch dir.
    tmp_dir = '_pex_tmp'
    os.mkdir(tmp_dir)
    if os.path.islink(args.interpreter) and os.readlink(args.interpreter) == 'python-exec2c':
        # Some distros have this intermediate binary; it messes things up for
        # pex which derefences it to this binary which can't be invoked directly.
        print('Can\'t determine Python interpreter; you should set the \n'
              'default_interpreter property in the [python] section of \n'
              'plzconfig.local to a specific version (e.g. /usr/bin/python3.4)')
        sys.exit(1)
    interpreter = PythonInterpreter.from_binary(args.interpreter)
    pex_builder = PEXBuilder(path=tmp_dir, interpreter=interpreter)
    pex_builder.info.zip_safe = args.zip_safe
    if args.shebang:
        pex_builder.set_shebang(args.shebang)
    # Override the entry point so the module import override works.
    pex_builder.info.entry_point = 'test_main' if args.test_srcs else 'pex_main'

    if args.scan:
        # Add everything under the input dir.
        add_directory(args.src_dir, '.', pex_builder, exclude=tmp_dir)
    else:
        # Just add bootstrap dir and main.
        add_directory(args.src_dir, '.bootstrap', pex_builder)
        pex_builder.add_source('pex_main.py', 'pex_main.py')

    # This function does some setuptools malarkey which is vexing me, so
    # I'm just gonna cheekily disable it for now.
    pex_builder._prepare_bootstrap = lambda: None
    # Similarly I don't want it to write __init__.py files, we will take care
    # of that later ourselves. This is important because we fail if we try to
    # add different versions of the same file to the final pex, but because we
    # only have a partial filetree here it may add some which don't appear until later.
    pex_builder._prepare_inits = lambda: None

    # Generate the PEX file.
    pex_builder.build(args.out)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=USAGE)
    parser.add_argument('--src_dir', required=True)
    parser.add_argument('--out', required=True)
    parser.add_argument('--entry_point', required=True)
    parser.add_argument('--interpreter', default=sys.executable)
    parser.add_argument('--test_package')
    parser.add_argument('--test_srcs')
    parser.add_argument('--shebang')
    parser.add_argument('--module_dir', default='')
    parser.add_argument('--zip_safe', dest='zip_safe', action='store_true')
    parser.add_argument('--nozip_safe', dest='zip_safe', action='store_false')
    parser.add_argument('--scan', dest='scan', action='store_true')
    parser.add_argument('--noscan', dest='scan', action='store_false')
    parser.add_argument('--bootstrap', dest='bootstrap', action='store_true')
    parser.set_defaults(zip_safe=True, compile=False, scan=True, bootstrap=False)
    main(parser.parse_args())
