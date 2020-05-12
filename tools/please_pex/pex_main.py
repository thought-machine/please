"""Zipfile entry point which supports auto-extracting itself based on zip-safety."""

from importlib import import_module
from zipfile import ZipFile, ZipInfo, is_zipfile
import os
import runpy
import sys


PY_VERSION = sys.version_info

if PY_VERSION.major >= 3:
    from importlib import machinery
else:
    import imp

if PY_VERSION >= (3, 2):
    from os import makedirs
else:
    # backported from cpython 3.8
    def makedirs(name, mode=0o777, exist_ok=False):
        """makedirs(name [, mode=0o777][, exist_ok=False])
        Super-mkdir; create a leaf directory and all intermediate ones.  Works like
        mkdir, except that any intermediate path segment (not just the rightmost)
        will be created if it does not exist. If the target directory already
        exists, raise an OSError if exist_ok is False. Otherwise no exception is
        raised.  This is recursive.
        """
        head, tail = os.path.split(name)
        if not tail:
            head, tail = os.path.split(head)
        if head and tail and not os.path.exists(head):
            try:
                makedirs(head, exist_ok=exist_ok)
            except FileExistsError:
                # Defeats race condition when another thread created the path
                pass
            cdir = curdir
            if isinstance(tail, bytes):
                cdir = bytes(curdir, "ASCII")
            if tail == cdir:  # xxx/newdir/. exists if xxx/newdir exists
                return
        try:
            os.mkdir(name, mode)
        except OSError:
            # Cannot rely on checking for EEXIST, since the operating system
            # could give priority to other errors like EACCES or EROFS
            if not exist_ok or not os.path.isdir(name):
                raise


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
PEX_STAMP = '__PEX_STAMP__'

# Workaround for https://bugs.python.org/issue15795
class ZipFileWithPermissions(ZipFile):
    """ Custom ZipFile class handling file permissions. """

    def _extract_member(self, member, targetpath, pwd):
        if not isinstance(member, ZipInfo):
            member = self.getinfo(member)

        targetpath = super(ZipFileWithPermissions, self)._extract_member(
            member, targetpath, pwd
        )

        attr = member.external_attr >> 16
        if attr != 0:
            os.chmod(targetpath, attr)
        return targetpath

class SoImport(object):
    """So import. Much binary. Such dynamic. Wow."""

    def __init__(self):

        if PY_VERSION.major < 3:
            self.suffixes = {x[0]: x for x in imp.get_suffixes() if x[2] == imp.C_EXTENSION}
        else:
            self.suffixes = machinery.EXTENSION_SUFFIXES  # list, as importlib will not be using the file description

        self.suffixes_by_length = sorted(self.suffixes, key=lambda x: -len(x))
        # Identify all the possible modules we could handle.
        self.modules = {}
        if is_zipfile(sys.argv[0]):
            zf = ZipFileWithPermissions(sys.argv[0])
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
            if PY_VERSION.major < 3:
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

    def find_distributions(self, context):
        """Return an iterable of all Distribution instances capable of
        loading the metadata for packages for the indicated ``context``.
        """

        try:
            from importlib_metadata import Distribution
            import re
        except:
            pass
        else:

            class PexDistribution(Distribution):
                template = r"{path}(-.*)?\.(dist|egg)-info/{filename}"

                def __init__(self, name, prefix=MODULE_DIR):
                    """Construct a distribution for a pex file to the metadata directory.

                    :param name: A module name
                    :param prefix: Modules prefix
                    """
                    self._name = name
                    self._prefix = prefix

                def _match_file(self, name, filename):
                    if re.match(
                        self.template.format(
                            path=os.path.join(self._prefix, self._name),
                            filename=filename,
                        ),
                        name,
                    ):
                        return name

                def read_text(self, filename):
                    if is_zipfile(sys.argv[0]):
                        zf = ZipFileWithPermissions(sys.argv[0])
                        for name in zf.namelist():
                            if name and self._match_file(name, filename):
                                return zf.read(name).decode(encoding="utf-8")

                read_text.__doc__ = Distribution.read_text.__doc__

                def _has_distribution(self):
                    if is_zipfile(sys.argv[0]):
                        zf = ZipFileWithPermissions(sys.argv[0])
                        for name in zf.namelist():
                            if name and self._match_file(name, ""):
                                return True

            if context.name in sys.modules:
                distribution = PexDistribution(context.name)
                if distribution._has_distribution():
                    yield distribution

    def get_code(self, fullname):
        module = self.load_module(fullname)
        return module.__loader__.get_code(fullname)


def pex_basepath(temp=False):
    if temp:
        import tempfile
        return tempfile.mkdtemp(dir=os.environ.get('TEMP_DIR'), prefix='pex_')
    else:
        return os.environ.get('PEX_CACHE_DIR',os.path.expanduser('~/.cache/pex'))


def pex_uniquedir():
    return 'pex-%s' % PEX_STAMP


def pex_paths():
    no_cache = os.environ.get('PEX_NOCACHE')
    no_cache = no_cache and no_cache.lower() == 'true'
    basepath, uniquedir = pex_basepath(no_cache), pex_uniquedir()
    pex_path = os.path.join(basepath, uniquedir)
    return pex_path, basepath, uniquedir, no_cache


def explode_zip():
    """Extracts the current pex to a temp directory where we can import everything from.

    This is primarily used for binary extensions which can't be imported directly from
    inside a zipfile.
    """
    # Temporarily add bootstrap to sys path
    sys.path = [os.path.join(sys.path[0], '.bootstrap')] + sys.path[1:]
    import contextlib, portalocker
    sys.path = sys.path[1:]

    @contextlib.contextmanager
    def pex_lockfile(basepath, uniquedir):
        # Acquire the lockfile.
        lockfile_path = os.path.join(basepath, '.lock-%s' % uniquedir)
        lockfile = open(lockfile_path, "a+")
        # Block until we can acquire the lockfile.
        portalocker.lock(lockfile, portalocker.LOCK_EX)
        lockfile.seek(0)
        yield lockfile
        portalocker.lock(lockfile, portalocker.LOCK_UN)

    @contextlib.contextmanager
    def _explode_zip():
        # We need to update the actual variable; other modules are allowed to look at
        # these variables to find out what's going on (e.g. are we zip-safe or not).
        global PEX_PATH

        PEX_PATH, basepath, uniquedir, no_cache = pex_paths()
        makedirs(basepath, exist_ok=True)
        with pex_lockfile(basepath, uniquedir) as lockfile:
            if len(lockfile.read()) == 0:
                import compileall, zipfile

                makedirs(PEX_PATH, exist_ok=True)
                with ZipFileWithPermissions(PEX, "r") as zf:
                    zf.extractall(PEX_PATH)

                if not no_cache:  # Don't bother optimizing; we're deleting this when we're done.
                    compileall.compile_dir(PEX_PATH, optimize=2, quiet=1)

                # Writing nonempty content to the lockfile will signal to subsequent invocations
                # that the cache has already been prepared.
                lockfile.write("pex unzip completed")
        sys.path = [PEX_PATH] + [x for x in sys.path if x != PEX]
        yield
        if no_cache:
            import shutil
            shutil.rmtree(basepath)

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
