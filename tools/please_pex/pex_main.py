"""Zipfile entry point which supports auto-extracting itself based on zip-safety."""

import importlib
import os
import runpy
import site
import sys

# Put this pex on the path before anything else.
PEX = os.path.abspath(sys.argv[0])
# This might get overridden down the line if the pex isn't zip-safe.
PEX_PATH = PEX
sys.path = [PEX_PATH] + sys.path

# These will get templated in by the build rules.
MODULE_DIR = '__MODULE_DIR__'
ENTRY_POINT = '__ENTRY_POINT__'
ZIP_SAFE = __ZIP_SAFE__


class ModuleDirImport(object):
    """Allows the given directory to be imported as though it was at the top level."""

    def __init__(self, package=MODULE_DIR):
        self.modules = set(self._listdir(package.replace('.', '/')))
        self.prefix = package + '.'

    def find_module(self, fullname, path=None):
        """Attempt to locate module. Returns self if found, None if not."""
        name, _, _ = fullname.partition('.')
        if name in self.modules:
            return self

    def load_module(self, fullname):
        mod = importlib.import_module(self.prefix + fullname)
        sys.modules[fullname] = mod
        return mod

    def _listdir(self, dirname):
        """Yields the contents of the given directory as Python modules."""
        import imp, zipfile
        suffixes = sorted([x[0] for x in imp.get_suffixes()], key=lambda x: -len(x))
        with zipfile.ZipFile(PEX, 'r') as zf:
            for name in zf.namelist():
                if name.startswith(dirname):
                    path, _ = self.splitext(name[len(dirname)+1:], suffixes)
                    if path:
                        path, _, _ = path.partition('/')
                        yield path.replace('/', '.')

    def splitext(self, path, suffixes):
        """Similar to os.path.splitext, but splits our longest known suffix preferentially."""
        for suffix in suffixes:
            if path.endswith(suffix):
                return path[:-len(suffix)], suffix
        return None, None


def override_import(package=MODULE_DIR):
    """Augments system importer to allow importing from the given module as though it were at the top level."""
    sys.meta_path.insert(0, ModuleDirImport(package))


def clean_sys_path():
    """Remove anything from sys.path that isn't either the pex or the main Python install dir.

    NB: *not* site-packages or dist-packages or any of that malarkey, just the place where
        we get the actual Python standard library packages from).
    This would be cleaner if we could suppress loading site in the first place, but that isn't
    as easy as all that to build into a pex, unfortunately.
    """
    site_packages = site.getsitepackages()
    sys.path = [x for x in sys.path if not any(x.startswith(pkg) for pkg in site_packages)]


def explode_zip():
    """Extracts the current pex to a temp directory where we can import everything from.

    This is primarily used for binary extensions which can't be imported directly from
    inside a zipfile.
    """
    import contextlib, shutil, tempfile, zipfile

    @contextlib.contextmanager
    def _explode_zip():
        # We need to update the actual variable; other modules are allowed to look at
        # these variables to find out what's going on (e.g. are we zip-safe or not).
        global PEX_PATH
        PEX_PATH = tempfile.mkdtemp(dir=os.environ.get('TEMP_DIR'), prefix='pex_')
        with zipfile.ZipFile(PEX, 'r') as zf:
            zf.extractall(PEX_PATH)
        # Strip the pex paths so nothing accidentally imports from there.
        sys.path = [PEX_PATH] + [x for x in sys.path if x != PEX]
        yield
        shutil.rmtree(PEX_PATH)

    return _explode_zip


def profile(filename):
    """Returns a context manager to perform profiling while the program runs.

    This is triggered by setting the PEX_PROFILE_FILENAME env var to the destination file,
    at which point this will be invoked automatically at pex startup.
    """
    import contextlib, cProfile

    @contextlib.contextmanager
    def _profile():
        profiler = cProfile.Profile()
        profiler.enable()
        yield
        profiler.disable()
        sys.stderr.write('Writing profiler output to %s\n' % filename)
        profiler.dump_stats(filename)

    return _profile


def main():
    """Runs the 'real' entry point of the pex.

    N.B. This gets redefined by test_main to run tests instead.
    """
    # Must run this as __main__ so it executes its own __name__ == '__main__' block.
    runpy.run_module(ENTRY_POINT, run_name='__main__')
    return 0  # unless some other exception gets raised, we're successful.
