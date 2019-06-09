"""Zipfile entry point which supports auto-extracting itself based on zip-safety."""

from importlib import import_module
import zipfile
import os
import runpy
import sys

PY_VERSION = int(sys.version[0])

if PY_VERSION >= 3:
    from importlib import machinery
else:
    import imp

try:
    from site import getsitepackages
except:
    def getsitepackages(prefixes=[sys.prefix, sys.exec_prefix]):
        """Returns a list containing all global site-packages directories.

        For each directory present in ``prefixes`` (or the global ``PREFIXES``),
        this function will find its `site-packages` subdirectory depending on the
        system environment, and will return a list of full paths.
        """
        sitepackages = []
        seen = set()

        if prefixes is None:
            prefixes = PREFIXES

        for prefix in prefixes:
            if not prefix or prefix in seen:
                continue
            seen.add(prefix)

            if os.sep == '/':
                sitepackages.append(os.path.join(prefix, "lib",
                                            "python%d.%d" % sys.version_info[:2],
                                            "site-packages"))
            else:
                sitepackages.append(prefix)
                sitepackages.append(os.path.join(prefix, "lib", "site-packages"))

        return sitepackages

# Put this pex on the path before anything else.
PEX = os.path.abspath(sys.argv[0])
# This might get overridden down the line if the pex isn't zip-safe.
PEX_PATH = PEX
sys.path = [PEX_PATH] + sys.path

# These will get templated in by the build rules.
MODULE_DIR = '__MODULE_DIR__'
ENTRY_POINT = '__ENTRY_POINT__'
ZIP_SAFE = __ZIP_SAFE__


class SoImport(object):
    """So import. Much binary. Such dynamic. Wow."""

    def __init__(self):

        if PY_VERSION < 3:
            self.suffixes = {x[0]: x for x in imp.get_suffixes() if x[2] == imp.C_EXTENSION}
        else:
            self.suffixes = machinery.EXTENSION_SUFFIXES  # list, as importlib will not be using the file description

        self.suffixes_by_length = sorted(self.suffixes, key=lambda x: -len(x))
        # Identify all the possible modules we could handle.
        self.modules = {}
        if zipfile.is_zipfile(sys.argv[0]):
            zf = zipfile.ZipFile(sys.argv[0])
            for name in zf.namelist():
                path, _ = self.splitext(name)
                if path:
                    if path.startswith('.bootstrap/'):
                        path = path[len('.bootstrap/'):]
                    importpath = path.replace('/', '.')
                    self.modules.setdefault(importpath, name)
                    if path.startswith(MODULE_DIR):
                        self.modules.setdefault(importpath[len(MODULE_DIR)+1:], name)
            if self.modules:
                self.zf = zf

    def find_module(self, fullname, path=None):
        """Attempt to locate module. Returns self if found, None if not."""
        if fullname in self.modules:
            return self

    def load_module(self, fullname):
        """Actually load a module that we said we'd handle in find_module."""
        import tempfile

        filename = self.modules[fullname]
        prefix, ext = self.splitext(filename)
        with tempfile.NamedTemporaryFile(suffix=ext, prefix=os.path.basename(prefix)) as f:
            f.write(self.zf.read(filename))
            f.flush()
            if PY_VERSION < 3:
                suffix = self.suffixes[ext]
                mod = imp.load_module(fullname, None, f.name, suffix)
            else:
                mod = machinery.ExtensionFileLoader(fullname, f.name).load_module()
        # Make it look like module came from the original location for nicer tracebacks.
        mod.__file__ = filename
        return mod

    def splitext(self, path):
        """Similar to os.path.splitext, but splits our longest known suffix preferentially."""
        for suffix in self.suffixes_by_length:
            if path.endswith(suffix):
                return path[:-len(suffix)], suffix
        return None, None


class ModuleDirImport(object):
    """Handles imports to a directory equivalently to them being at the top level.

    This means that if one writes `import third_party.python.six`, it's imported like `import six`,
    but becomes accessible under both names. This handles both the fully-qualified import names
    and packages importing as their expected top-level names internally.
    """

    def __init__(self, module_dir=MODULE_DIR):
        self.prefix = module_dir.replace('/', '.') + '.'

    def find_module(self, fullname, path=None):
        """Attempt to locate module. Returns self if found, None if not."""
        if fullname.startswith(self.prefix):
            return self

    def load_module(self, fullname):
        """Actually load a module that we said we'd handle in find_module."""
        module = import_module(fullname[len(self.prefix):])
        sys.modules[fullname] = module
        return module

    def get_code(self, fullname):
        module = self.load_module(fullname)
        return module.__loader__.get_code(fullname)


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
        if not os.environ.get('PEX_SAVE_TEMP_DIR'):
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


def interact(main):
    """If PEX_INTERPRETER is set, then starts an interactive console, otherwise runs main()."""
    if os.environ.get('PEX_INTERPRETER', '0') != '0':
        import code
        code.interact()
    else:
        return main()


def main():
    """Runs the 'real' entry point of the pex.

    N.B. This gets redefined by test_main to run tests instead.
    """
    # Must run this as __main__ so it executes its own __name__ == '__main__' block.
    runpy.run_module(ENTRY_POINT, run_name='__main__')
    return 0  # unless some other exception gets raised, we're successful.
