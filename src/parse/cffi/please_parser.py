# Wrapper script invoked from Please to parse build files.

try:
    import __builtin__ as builtins
    is_py3 = False
except ImportError:
    import builtins
    is_py3 = True
import ast
import imp
import os
from collections import defaultdict, Mapping
from contextlib import contextmanager
from types import FunctionType
from parser_interface import ffi


_please_builtins = imp.new_module('_please_builtins')
_please_globals = _please_builtins.__dict__
_keepalive_functions = set()
_build_code_cache = {}
_c_subinclude_package_name = None
_subinclude_package_name = None
_subinclude_package = None

# List of everything we keep in the builtins module. This is a pretty agricultural way
# of restricting what build files can do - no doubt there'd be clever ways of working
# around it - but at least it will give people the sense that they shouldn't use some of these.
# We also implicitly keep all the exception types.
_WHITELISTED_BUILTINS = {
    'None', 'False', 'True', 'abs', 'all', 'any', 'apply', 'basestring', 'bin', 'bool',
    'callable', 'chr', 'classmethod', 'cmp', 'coerce', 'complex', 'delattr', 'dict', 'dir',
    'divmod', 'enumerate', 'filter', 'float', 'format', 'frozenset', 'getattr', 'globals',
    'hasattr', 'hash', 'hex', 'id', 'input', 'int', 'isinstance', 'issubclass', 'iter',
    'len', 'list', 'locals', 'long', 'map', 'max', 'min', 'next', 'object', 'oct', 'ord',
    'bytearray', 'pow', 'print', 'property', 'range', 'reduce', 'repr', 'reversed', 'round',
    'sequenceiterator', 'set', 'setattr', 'slice', 'sorted', 'staticmethod', 'str', 'sum',
    'super', 'tuple', 'type', 'unichr', 'unicode', 'vars', 'zip', '__name__',
    'NotImplemented',
    'compile', '__import__',  # We disallow importing separately, it's too hard to do here
    '__cffi_backend_extern_py',  # This gets added with cpython / cffi 1.6+ and is pretty crucial.
}
if is_py3:
    _WHITELISTED_BUILTINS |= {'bytes', 'exec'}  # These are needed too
    # Have to be more careful with encodings in python 3.
    ffi_to_string = lambda c: ffi.string(c).decode('utf-8')
    ffi_from_string = lambda s: ffi.new('char[]', s.encode('utf-8'))
else:
    ffi_to_string = ffi.string
    ffi_from_string = lambda s: ffi.new('char[]', s)
    builtins.range = builtins.xrange

# Used to indicate that parsing of a target is deferred because it requires another target.
_DEFER_PARSE = '_DEFER_'
_FFI_DEFER_PARSE = ffi_from_string(_DEFER_PARSE)


@ffi.def_extern('ParseFile')
def parse_file(c_filename, c_package_name, c_package):
    try:
        filename = ffi_to_string(c_filename)
        package_name = ffi_to_string(c_package_name)
        builtins = _get_globals(c_package, c_package_name)
        _parse_build_code(filename, builtins)
        return ffi.NULL
    except DeferParse as err:
        return _FFI_DEFER_PARSE
    except Exception as err:
        return ffi_from_string(str(err))


@ffi.def_extern('ParseCode')
def parse_code(c_code, c_filename, c_package):
    if c_package != 0:
        global _subinclude_package, _subinclude_package_name, _c_subinclude_package_name
        _subinclude_package_name = ffi_to_string(c_filename)
        _c_subinclude_package_name = c_filename
        _subinclude_package = c_package
        return ffi.NULL
    try:
        filename = ffi_to_string(c_filename)
        code = ffi_to_string(c_code)
        # Note we don't go through _parse_build_code - there's no need to perform the ast
        # walk on code that we control internally. This conceptually means that we *could*
        # import in those files, but we will not do that because it would be sheer peasantry.
        code = _compile(code, filename, 'exec')
        exec(code, _please_globals)
        return ffi.NULL
    except Exception as err:
        return ffi_from_string(str(err))


@ffi.def_extern('RunCode')
def run_code(c_code):
    """Executes some arbitrary code in a normal Python context.

    This isn't made available during parse time, it's for some internal functionality.
    """
    try:
        code = _compile(ffi_to_string(c_code), '<data>', 'exec')
        exec(code, globals())
        return ffi.NULL
    except Exception as err:
        return ffi_from_string(str(err))


def _parse_build_code(filename, globals_dict, cache=False):
    """Parses given file and interprets it. Optionally caches code for future reuse."""
    code = _build_code_cache.get(filename)
    if not code:
        with _open(filename) as f:
            tree = ast.parse(f.read(), filename)
        for node in ast.iter_child_nodes(tree):
            if isinstance(node, ast.Import) or isinstance(node, ast.ImportFrom):
                raise SyntaxError('import not allowed')
            if not is_py3 and isinstance(node, ast.Exec):
                raise SyntaxError('exec not allowed')
            if not is_py3 and isinstance(node, ast.Print):
                raise SyntaxError('print not allowed, use log functions instead')
        code = _compile(tree, filename, 'exec')
        if cache:
            _build_code_cache[filename] = code
    exec(code, globals_dict)


def bazel_wrapper(func):
    """Rewrites incoming argument names when we're in Bazel compatibility mode."""
    def _inner(*args, **kwargs):
        for k, v in _BAZEL_KEYWORD_REWRITES.items():
            if k in kwargs:
                if v in kwargs:
                    raise ValueError('You must pass at most one of %s and %s' % (k, v))
                kwargs[v] = kwargs[k]
                del kwargs[k]
        return func(*args, **kwargs)
    return _inner


_BAZEL_KEYWORD_REWRITES = {
    'artifact': 'id',
    'copts': 'compiler_flags',
    'linkopts': 'linker_flags',
    'testonly': 'test_only',
    'javacopts': 'javac_flags',
    'tags': 'labels',
    'runtime_deps': 'data',
    'exports': 'exported_deps',
}


@ffi.def_extern('SetConfigValue')
def set_config_value(c_name, c_value):
    name = ffi_to_string(c_name)
    value = ffi_to_string(c_value)
    config = _please_globals['CONFIG']
    existing = config.get(name)
    # A little gentle hack to make it convenient to set repeated config values; we could
    # do it via another callback but we already have so many of them...
    if isinstance(existing, list):
        existing.append(value)
    elif existing:
        config[name] = [existing, value]
    else:
        config[name] = value


def include_defs(package, dct, target):
    filename = ffi_to_string(_get_include_file(package, ffi_from_string(target)))
    # Dodgy in-band signalling of errors follows.
    if filename.startswith('__'):
        raise ParseError(filename.lstrip('_'))
    _parse_build_code(filename, dct, cache=True)


def subinclude(package, dct, target, hash=None):
    """Includes the output of a build target as extra rules in this one."""
    if target.startswith('http'):
        target = _get_subinclude_target(target, hash)
    filename = ffi_to_string(_get_subinclude_file(package, ffi_from_string(target)))
    if filename == _DEFER_PARSE:
        raise DeferParse(filename)
    elif filename.startswith('__'):
        raise ParseError(filename.lstrip('_'))
    _parse_build_code(filename, dct, cache=True)


def _get_subinclude_target(url, hash):
    """Creates a remote_file target to subinclude() a remote url and returns its name."""
    name = os.path.basename(url).replace('.', '_')
    try:
        _get_globals(_subinclude_package, _c_subinclude_package_name).get('remote_file')(
            name = name,
            url = url,
            hashes = [hash] if hash else [],
            visibility = ['PUBLIC'],
        )
    except DuplicateTargetError:
        pass  # Bit dodgy but assume it's already added.
    return '//%s:%s' % (_subinclude_package_name, name)


def build_rule(globals_dict, package, name, cmd, test_cmd=None, srcs=None, data=None, outs=None,
               deps=None, exported_deps=None, tools=None, labels=None, visibility=None, hashes=None,
               binary=False, test=False, test_only=None, building_description='Building...',
               needs_transitive_deps=False, output_is_complete=False, container=False,
               no_test_output=False, flaky=0, build_timeout=0, test_timeout=0,
               pre_build=None, post_build=None, requires=None, provides=None, licences=None,
               test_outputs=None, system_srcs=None, stamp=False, tag='', optional_outs=None,
               _filegroup=False):
    if name == 'all':
        raise ValueError('"all" is a reserved build target name.')
    if '/' in name or ':' in name:
        raise ValueError(': and / are reserved characters in build target names')
    if container and not test:
        raise ValueError('Only tests can have container=True')
    if test_cmd and not test:
        raise ValueError('Target %s has been given a test command but isn\'t a test' % name)
    if tag:
        name = ''.join(['_' if not name.startswith('_') else '',
                        name,
                        '_' if '#' in name else '#',
                        tag])
    if not _is_valid_target_name(ffi_from_string(name)):
        raise ValueError('"%s" is not a valid target name' % name)
    if visibility is None:
        visibility = globals_dict['CONFIG'].get('DEFAULT_VISIBILITY')
    if licences is None:
        licences = globals_dict['CONFIG'].get('DEFAULT_LICENCES')
    if test_only is None:
        test_only = globals_dict['CONFIG'].get('DEFAULT_TESTONLY')

    # Further calls to package() are now banned; it's too difficult to ensure pre/post build
    # functions work as expected if the user changes things after adding the target but before
    # said function runs.
    globals_dict['package'] = package_banned

    ffi_string = lambda x: ffi.NULL if x is None else ffi_from_string(x)
    target = _add_target(package,
                         ffi_string(name),
                         ffi_string('' if isinstance(cmd, Mapping) else cmd.strip()),
                         ffi_string('' if isinstance(test_cmd, Mapping) else test_cmd.strip() if test_cmd else None),
                         binary,
                         test,
                         needs_transitive_deps,
                         output_is_complete,
                         bool(container),
                         no_test_output,
                         test_only or test,  # Tests are implicitly test_only
                         stamp,
                         _filegroup,
                         3 if flaky is True else flaky,  # Default is to rerun three times.
                         build_timeout,
                         test_timeout,
                         ffi_string(building_description))
    if not target:
        # Currently this is the only reason _add_target can fail, given that we validated
        # the target name earlier. Bit hacky but will have to do for now.
        raise DuplicateTargetError('Duplicate target %s' % name)
    if isinstance(srcs, Mapping):
        for src_name, src_list in srcs.items():
            if isinstance(src_list, str):
                raise ValueError('Value in named_srcs for target %s is a string, you probably '
                                 'meant to use a list of strings instead' % name)
            elif src_list:
                for src in src_list:
                    _check_c_error(_add_named_src(target, src_name, src))
    elif srcs:
        for src in srcs:
            if src and src.startswith('/') and not src.startswith('//'):
                raise ValueError('Entry "%s" in srcs of %s has an absolute path; that\'s not allowed. '
                                 'You might want to try system_srcs instead' % (src, name))
        _add_strings(target, _add_src, srcs, 'srcs')
    if isinstance(cmd, Mapping):
        for config, command in cmd.items():
            _check_c_error(_add_command(target, config, command.strip()))
    if isinstance(test_cmd, Mapping):
        for config, command in test_cmd.items():
            _check_c_error(_add_test_command(target, config, command.strip()))
    if system_srcs:
        for src in system_srcs:
            if not src.startswith('/') or src.startswith('//'):
                raise ValueError('Entry "%s" in system_srcs of %s is not an absolute path. '
                                 'You might want to try srcs instead' % (src, name))
        _add_strings(target, _add_src, system_srcs, 'system_srcs')
    _add_strings(target, _add_data, data, 'data')
    _add_strings(target, _add_dep, deps, 'deps')
    _add_strings(target, _add_exported_dep, exported_deps, 'exported_deps')
    _add_strings(target, _add_tool, tools, 'tools')
    _add_strings(target, _add_out, outs, 'outs')
    _add_strings(target, _add_optional_out, optional_outs, 'optional_outs')
    _add_strings(target, _add_vis, visibility, 'visibility')
    _add_strings(target, _add_label, labels, 'labels')
    _add_strings(target, _add_hash, hashes, 'hashes')
    _add_strings(target, _add_licence, licences, 'licences')
    _add_strings(target, _add_test_output, test_outputs, 'test_outputs')
    _add_strings(target, _add_require, requires, 'requires')
    if provides:
        if not isinstance(provides, Mapping):
            raise ValueError('"provides" argument for rule %s is not a mapping' % name)
        for lang, rule in provides.items():
            _check_c_error(_add_provide(target, ffi_from_string(lang), ffi_from_string(rule)))
    if pre_build:
        # Must manually ensure we keep these objects from being gc'd.
        handle = ffi.new_handle(pre_build)
        _keepalive_functions.add(pre_build)
        _keepalive_functions.add(handle)
        _set_pre_build_callback(handle, pre_build.__code__.co_code, target)
    if post_build:
        handle = ffi.new_handle(post_build)
        _keepalive_functions.add(post_build)
        _keepalive_functions.add(handle)
        _set_post_build_callback(handle, post_build.__code__.co_code, target)
    if isinstance(container, dict):
        for k, v in container.items():
            _set_container_setting(target, k, v)
    return ':' + name


@ffi.def_extern('PreBuildFunctionRunner')
def run_pre_build_function(handle, package, name):
    try:
        callback = ffi.from_handle(handle)
        callback(ffi_to_string(name))
        return ffi.NULL
    except DeferParse:
        return ffi_from_string("Don't try to subinclude() from inside a pre-build function")
    except Exception as err:
        return ffi_from_string(str(err))


@ffi.def_extern('PostBuildFunctionRunner')
def run_post_build_function(handle, package, name, output):
    try:
        callback = ffi.from_handle(handle)
        callback(ffi_to_string(name), ffi_to_string(output).strip().split('\n'))
        return ffi.NULL
    except DeferParse:
        return ffi_from_string("Don't try to subinclude() from inside a post-build function")
    except Exception as err:
        return ffi_from_string(str(err))


def _add_strings(target, func, lst, name):
    if lst:
        for x in lst:
            if x:
                _check_c_error(func(target, ffi_from_string(x)))


def _check_c_error(error):
    """Converts returned errors from cffi to exceptions."""
    if error:
        raise ParseError(ffi_to_string(error))


def glob(package, includes, excludes=None, exclude=None, hidden=False):
    if isinstance(includes, str):
        raise TypeError('The first argument to glob() should be a list')
    excludes = excludes or exclude
    includes_keepalive = [ffi_from_string(include) for include in includes]
    excludes_keepalive = [ffi_from_string(exclude) for exclude in excludes or []]
    filenames = _glob(ffi_from_string(package),
                      ffi.new('char*[]', includes_keepalive),
                      len(includes_keepalive),
                      ffi.new('char*[]', excludes_keepalive),
                      len(excludes_keepalive),
                      hidden)
    return [ffi_to_string(filename) for filename in _null_terminated_array(filenames)]


def get_labels(package, target, prefix):
    """Gets the transitive set of labels for a rule. Should be called from a pre-build function."""
    labels = _get_labels(package, ffi_from_string(target), ffi_from_string(prefix))
    return [ffi_to_string(label) for label in _null_terminated_array(labels)]


def has_label(package, target, prefix):
    """Returns True if the target has any matching label that would be returned by get_labels."""
    return bool(get_labels(package, target, prefix))


def package(globals_dict, **kwargs):
    """Defines settings affecting the current package - for example, default visibility."""
    config = globals_dict['CONFIG'].copy()
    for k, v in kwargs.items():
        k = k.upper()
        if k in config:
            config[k] = v
        else:
            raise KeyError('error calling package(): %s is not a known config value' % k)
    globals_dict['CONFIG'] = config


def package_banned(*args, **kwargs):
    """Replaces package() after the first target is added."""
    raise ParseError("package() must be called before any build targets are defined")


def licenses(globals_dict, licenses):
    """Defines default licenses for the package. Provided for Bazel compatibility."""
    package(globals_dict, default_licences=licenses)


def _null_terminated_array(arr):
    for i in range(1000000):
        if arr[i] == ffi.NULL:
            break
        yield arr[i]


def _get_globals(c_package, c_package_name):
    """Creates a copy of the builtin set of globals to use on interpreting new files.

    Best not to ask about any of this really. If you must know: all Python functions store their
    own set of globals internally, which we want to change to point to this local dict so it's
    indistinguishable from before. It's not sufficient just to update their __globals__ and you
    can't reassign that at runtime, so we create duplicates here. YOLO.
    """
    local_globals = {}
    bazel_compat = _please_globals.get('CONFIG', {}).get('BAZEL_COMPATIBILITY')
    for k, v in _please_globals.items():
        if callable(v) and type(v) == FunctionType:
            func = FunctionType(v.__code__, local_globals, k, v.__defaults__, v.__closure__)
            local_globals[k] = bazel_wrapper(func) if bazel_compat else func
        else:
            local_globals[k] = v
    # Need to pass some hidden arguments to these guys.
    package_name = ffi_to_string(c_package_name)
    local_globals['subinclude'] = lambda *args, **kwargs: subinclude(c_package, local_globals, *args, **kwargs)
    local_globals['build_rule'] = lambda *args, **kwargs: build_rule(local_globals, c_package, *args, **kwargs)
    local_globals['glob'] = lambda *args, **kwargs: glob(package_name, *args, **kwargs)
    local_globals['get_labels'] = lambda name, prefix: get_labels(c_package, name, prefix)
    local_globals['has_label'] = lambda name, prefix: has_label(c_package, name, prefix)
    local_globals['get_base_path'] = lambda: package_name
    local_globals['add_dep'] = lambda target, dep: _check_c_error(_add_dependency(c_package, target, dep, False))
    local_globals['add_exported_dep'] = lambda target, dep: _check_c_error(_add_dependency(c_package, target, dep, True))
    local_globals['add_out'] = lambda target, out: _check_c_error(_add_output(c_package, target, out))
    local_globals['add_licence'] = lambda name, licence: _check_c_error(_add_licence_post(c_package, name, licence))
    local_globals['get_command'] = lambda name, config='': ffi_to_string(_get_command(c_package, name, config))
    local_globals['set_command'] = lambda name, config, command='': _check_c_error(_set_command(c_package, name, config, command))
    local_globals['package'] = lambda **kwargs: package(local_globals, **kwargs)
    # Make these available to other scripts so they can get it without import.
    local_globals['join_path'] = os.path.join
    local_globals['split_path'] = os.path.split
    local_globals['splitext'] = os.path.splitext
    local_globals['basename'] = os.path.basename
    local_globals['dirname'] = os.path.dirname
    # The levels here are internally interpreted to match go-logging's levels.
    local_globals['log'] = DotDict({
        'fatal': lambda message, *args: _log(0, c_package, message % args),
        'error': lambda message, *args: _log(1, c_package, message % args),
        'warning': lambda message, *args: _log(2, c_package, message % args),
        'notice': lambda message, *args: _log(3, c_package, message % args),
        'info': lambda message, *args: _log(4, c_package, message % args),
        'debug': lambda message, *args: _log(5, c_package, message % args),
    })
    if bazel_compat:
        # include_defs is used indirectly. It's also nice to switch this on for limited Buck compatibility too.
        local_globals['include_defs'] = lambda *args, **kwargs: include_defs(c_package, local_globals, *args, **kwargs)
        local_globals['native'] = DotDict(local_globals)
        local_globals['licenses'] = lambda l: licenses(local_globals, l)
        local_globals['PACKAGE_NAME'] = package_name
    return local_globals


@ffi.def_extern('RegisterCallback')
def register_callback(name, c_type, callback):
    """Called at initialisation time to register a single callback."""
    f = ffi.cast(ffi_to_string(c_type), callback)
    if is_py3:
        # Wrap the function up to auto-encode to bytes (ffi requires this in py3)
        # TODO(pebers): this is not exactly beautiful, can we find a better way of handling it?
        globals()[ffi_to_string(name)] = lambda *args: f(
            *[ffi_from_string(arg) if isinstance(arg, str) else arg for arg in args])
    else:
        globals()[ffi_to_string(name)] = f
    return 1  # used to detect success (must be nonzero)


class ParseError(Exception):
    """Raised on general file parsing errors."""


class DuplicateTargetError(ParseError):
    """Raised when a duplicate target is added."""


class DeferParse(Exception):
    """Raised to include that the parse of a file will be deferred until some build actions are done."""


# Derive to support dot notation.
class DotDict(dict):
    def __getattr__(self, attr):
        return self[attr]

    def copy(self):
        return DotDict(self)

_please_globals['CONFIG'] = DotDict()
_please_globals['CONFIG']['DEFAULT_VISIBILITY'] = None
_please_globals['CONFIG']['DEFAULT_LICENCES'] = None
_please_globals['CONFIG']['DEFAULT_TESTONLY'] = False
_please_globals['defaultdict'] = defaultdict
_please_globals['ParseError'] = ParseError
_please_globals['DuplicateTargetError'] = DuplicateTargetError

# We'll need these guys locally. Unfortunately exec is a statement so we
# can't do it for that.
_compile, _open = compile, open
for k, v in list(builtins.__dict__.items()):  # YOLO
    try:
        if issubclass(v, BaseException):
            continue
    except:
        pass
    if k not in _WHITELISTED_BUILTINS:
        del builtins.__dict__[k]
