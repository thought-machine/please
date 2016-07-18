"""Customised pex entry point which forces imports of third-party components to a given directory."""

import os
import pkg_resources
import runpy
import sys

try:
    import builtins  # python 3
except ImportError:
    import __builtin__ as builtins  # python 2


# These will get templated in by the build rules.
MODULE_DIR = '__MODULE_DIR__'
ENTRY_POINT = '__ENTRY_POINT__'
ZIP_SAFE = __ZIP_SAFE__

ABSOLUTE_IMPORT_ONLY = 0
DEFAULT_IMPORT_LEVEL = -1 if sys.version_info[0] < 3 else 0


def override_import(package=MODULE_DIR):
    """Overrides builtin __import__ function to forcibly add the given directory.

    Returns an appropriate replacement for the builtin function which imports known
    third party modules as eg. 'third_party.python.six' instead of just 'six'.
    """
    if not package:
        return
    original_import = builtins.__import__
    try:
        modules = {(x.rpartition('.')[0] or x): p for p in package.split(',')
                   for x in pkg_resources.resource_listdir(p, '') if not x.startswith('__init__')}
    except ImportError:
        return  # Skip if the module isn't built into this pex
    if not modules:
        return  # nothing to do

    def _override_import(name, globals=None, locals=None, fromlist=None, level=DEFAULT_IMPORT_LEVEL):
        module_name, _, _ = name.partition('.')
        if module_name in modules and level < 1:
            prefix = modules[module_name] + '.'
            fq_name = prefix + name
            mod = original_import(fq_name, globals, locals, fromlist, level=ABSOLUTE_IMPORT_ONLY)
            if fromlist:
                return mod
            else:
                # Have to be careful to return the correct module here.
                # See http://stackoverflow.com/questions/2724260 if you're curious.
                module_name = name.partition('.')[0]
                mod = sys.modules[prefix + module_name]
            sys.modules[name] = sys.modules[prefix + name]
            return mod
        return original_import(name, globals, locals, fromlist, level)

    builtins.__import__ = _override_import


def clean_sys_path():
    """Remove anything from sys.path that isn't either the pex or the main Python install dir.

    NB: *not* site-packages or dist-packages or any of that malarkey, just the place where
        we get the actual Python standard library packages from).
    """
    sys.path = [x for x in sys.path if 'dist-packages' not in x and
                ('.pex' in x or 'please_pex' in x or x.startswith(os.path.split(os.__file__)[0]))]
    if not ZIP_SAFE:
        # Strip the pex paths if we're not zip safe so nothing accidentally imports from there.
        sys.path = [x for x in sys.path if not x.endswith('.pex')]


if __name__ == '__main__':
    override_import()
    clean_sys_path()
    # Must run this as __main__ so it executes its own __name__ == '__main__' block.
    runpy.run_module(ENTRY_POINT, run_name='__main__')
