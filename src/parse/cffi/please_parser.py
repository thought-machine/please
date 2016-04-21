# Wrapper script invoked from Please to parse build files.

import __builtin__
import ast
import imp
import os
from collections import defaultdict, Mapping
from contextlib import contextmanager
from types import FunctionType


_please_builtins = imp.new_module('_please_builtins')
_please_globals = _please_builtins.__dict__
_keepalive_functions = set()
_build_code_cache = {}

# List of everything we keep in the __builtin__ module. This is a pretty agricultural way
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
    'super', 'tuple', 'type', 'unichr', 'unicode', 'vars', 'xrange', 'zip', '__name__',
    'NotImplemented',
    'compile', '__import__',  # We disallow importing separately, it's too hard to do here
}

# Used to indicate that parsing of a target is deferred because it requires another target.
_DEFER_PARSE = '_DEFER_'
_FFI_DEFER_PARSE = ffi.new('char[]', _DEFER_PARSE)


@ffi.callback('ParseFileCallback*')
def parse_file(c_filename, c_package_name, c_package):
    try:
        filename = ffi.string(c_filename)
        package_name = ffi.string(c_package_name)
        builtins = _get_globals(c_package, c_package_name)
        _parse_build_code(filename, builtins)
        return ffi.NULL
    except DeferParse as err:
        return _FFI_DEFER_PARSE
    except Exception as err:
        return ffi.new('char[]', str(err))


@ffi.callback('ParseFileCallback*')
def parse_code(c_code, c_filename, _):
    try:
        code = ffi.string(c_code)
        filename = ffi.string(c_filename)
        # Note we don't go through _parse_build_code - there's no need to perform the ast
        # walk on code that we control internally. This conceptually means that we *could*
        # import in those files, but we will not do that because it would be sheer peasantry.
        code = _compile(code, filename, 'exec')
        exec(code, _please_globals)
        return ffi.NULL
    except Exception as err:
        return ffi.new('char[]', str(err))


def _parse_build_code(filename, globals_dict, cache=False):
    """Parses given file and interprets it. Optionally caches code for future reuse."""
    code = _build_code_cache.get(filename)
    if not code:
        with _open(filename) as f:
            tree = ast.parse(f.read(), filename)
        for node in ast.iter_child_nodes(tree):
            if isinstance(node, ast.Import) or isinstance(node, ast.ImportFrom):
                raise SyntaxError('import not allowed')
            if isinstance(node, ast.Exec):
                raise SyntaxError('exec not allowed')
            if isinstance(node, ast.Print):
                raise SyntaxError('print not allowed, use log functions instead')
        code = _compile(tree, filename, 'exec')
        _build_code_cache[filename] = code
    exec(code, globals_dict)


@ffi.callback('SetConfigValueCallback*')
def set_config_value(c_name, c_value):
    name = ffi.string(c_name)
    value = ffi.string(c_value)
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
    _log(2, package, 'include_defs is deprecated, use subinclude() instead')
    filename = ffi.string(_get_include_file(package, ffi.new('char[]', target)))
    # Dodgy in-band signalling of errors follows.
    if filename.startswith('__'):
        raise ParseError(filename.lstrip('_'))
    _parse_build_code(filename, dct, cache=True)


def subinclude(package, dct, target):
    """Includes the output of a build target as extra rules in this one."""
    filename = ffi.string(_get_subinclude_file(package, ffi.new('char[]', target)))
    if filename == _DEFER_PARSE:
        raise DeferParse(filename)
    elif filename.startswith('__'):
        raise ParseError(filename.lstrip('_'))
    with _open(filename) as f:
        code = _compile(f.read(), filename, 'exec')
    exec(code, dct)


def build_rule(globals_dict, package, name, cmd, test_cmd=None, srcs=None, data=None, outs=None,
               deps=None, exported_deps=None, tools=None, labels=None, visibility=None, hashes=None,
               binary=False, test=False, test_only=False, building_description='Building...',
               needs_transitive_deps=False, output_is_complete=False, container=False,
               skip_cache=False, no_test_output=False, flaky=0, build_timeout=0, test_timeout=0,
               pre_build=None, post_build=None, requires=None, provides=None, licences=None,
               test_outputs=None, system_srcs=None):
    if name == 'all':
        raise ValueError('"all" is a reserved build target name.')
    if '/' in name or ':' in name:
        raise ValueError(': and / are reserved characters in build target names')
    if container and not test:
        raise ValueError('Only tests can have container=True')
    if test_cmd and not test:
        raise ValueError('Target %s has been given a test command but isn\'t a test' % name)
    if test and not test_cmd:
        raise ValueError('Target %s is a test but hasn\'t been given a test command' % name)
    if visibility is None:
        visibility = globals_dict['CONFIG'].get('DEFAULT_VISIBILITY')
    if licences is None:
        licences = globals_dict['CONFIG'].get('DEFAULT_LICENCES')
    ffi_string = lambda x: ffi.NULL if x is None else ffi.new('char[]', x)
    target = _add_target(package,
                         ffi_string(name),
                         ffi_string('' if isinstance(cmd, Mapping) else cmd.strip()),
                         ffi_string(test_cmd),
                         binary,
                         test,
                         needs_transitive_deps,
                         output_is_complete,
                         bool(container),
                         no_test_output,
                         skip_cache,
                         test_only or test,  # Tests are implicitly test_only
                         3 if flaky is True else flaky,  # Default is to rerun three times.
                         build_timeout,
                         test_timeout,
                         ffi_string(building_description))
    if not target:
        raise ParseError('Failed to add target %s' % name)
    if isinstance(srcs, Mapping):
        for name, src_list in srcs.iteritems():
            if src_list:
                for src in src_list:
                    _check_c_error(_add_named_src(target, name, src))
    elif srcs:
        for src in srcs:
            if src.startswith('/') and not src.startswith('//'):
                raise ValueError('Entry "%s" in srcs of %s has an absolute path; that\'s not allowed. '
                                 'You might want to try system_srcs instead' % (src, name))
        _add_strings(target, _add_src, srcs, 'srcs')
    if isinstance(cmd, Mapping):
        for config, command in cmd.items():
            _check_c_error(_add_command(target, config, command.strip()))
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
            _check_c_error(_add_provide(target, ffi.new('char[]', lang), ffi.new('char[]', rule)))
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


@ffi.callback('PreBuildCallbackRunner*')
def run_pre_build_function(handle, package, name):
    try:
        callback = ffi.from_handle(handle)
        callback(ffi.string(name))
        return ffi.NULL
    except DeferParse:
        return ffi.new('char[]', "Don't try to subinclude() from inside a pre-build function")
    except Exception as err:
        return ffi.new('char[]', str(err))


@ffi.callback('PostBuildCallbackRunner*')
def run_post_build_function(handle, package, name, output):
    try:
        callback = ffi.from_handle(handle)
        callback(ffi.string(name), ffi.string(output).strip().split('\n'))
        return ffi.NULL
    except DeferParse:
        return ffi.new('char[]', "Don't try to subinclude() from inside a post-build function")
    except Exception as err:
        return ffi.new('char[]', str(err))


def _add_strings(target, func, lst, name):
    if lst:
        if isinstance(lst, str):
            # We don't want to enforce this is a list (any sequence should be fine) but it's
            # easy to use a string by mistake, which tends to cause some weird cffi errors later.
            raise ValueError('"%s" argument should be a list of strings, not a string' % name)
        for x in lst:
            _check_c_error(func(target, ffi.new('char[]', x)))


def _check_c_error(error):
    """Converts returned errors from cffi to exceptions."""
    if error:
        raise ParseError(ffi.string(error))


def glob(package, includes, excludes=None, hidden=False):
    if isinstance(includes, str):
        raise TypeError('The first argument to glob() should be a list')
    includes_keepalive = [ffi.new('char[]', include) for include in includes]
    excludes_keepalive = [ffi.new('char[]', exclude) for exclude in excludes or []]
    filenames = _glob(ffi.new('char[]', package),
                      ffi.new('char*[]', includes_keepalive),
                      len(includes_keepalive),
                      ffi.new('char*[]', excludes_keepalive),
                      len(excludes_keepalive),
                      hidden)
    return [ffi.string(filename) for filename in _null_terminated_array(filenames)]


def get_labels(package, target, prefix):
    """Gets the transitive set of labels for a rule. Should be called from a pre-build function."""
    labels = _get_labels(package, ffi.new('char[]', target), ffi.new('char[]', prefix))
    return [ffi.string(label) for label in _null_terminated_array(labels)]


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


def licenses(globals_dict, licenses):
    """Defines default licenses for the package. Provided for Bazel compatibility."""
    package(globals_dict, default_licences=licenses)


def _null_terminated_array(arr):
    for i in xrange(1000000):
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
    for k, v in _please_globals.iteritems():
        if callable(v) and type(v) == FunctionType:
            local_globals[k] = FunctionType(v.__code__, local_globals, k, v.__defaults__, v.__closure__)
        else:
            local_globals[k] = v
    # Need to pass some hidden arguments to these guys.
    package_name = ffi.string(c_package_name)
    local_globals['include_defs'] = lambda target: include_defs(c_package, local_globals, target)
    local_globals['subinclude'] = lambda target: subinclude(c_package, local_globals, target)
    local_globals['build_rule'] = lambda *args, **kwargs: build_rule(local_globals, c_package,
                                                                     *args, **kwargs)
    local_globals['glob'] = lambda *args, **kwargs: glob(package_name, *args, **kwargs)
    local_globals['get_labels'] = lambda name, prefix: get_labels(c_package, name, prefix)
    local_globals['has_label'] = lambda name, prefix: has_label(c_package, name, prefix)
    local_globals['get_base_path'] = lambda: package_name
    local_globals['add_dep'] = lambda target, dep: _check_c_error(_add_dependency(c_package, target, dep, False))
    local_globals['add_exported_dep'] = lambda target, dep: _check_c_error(_add_dependency(c_package, target, dep, True))
    local_globals['add_out'] = lambda target, out: _check_c_error(_add_output(c_package, target, out))
    local_globals['add_licence'] = lambda name, licence: _check_c_error(_add_licence_post(c_package, name, licence))
    local_globals['set_command'] = lambda name, config, command='': _check_c_error(_set_command(c_package, name, config, command))
    local_globals['package'] = lambda **kwargs: package(local_globals, **kwargs)
    local_globals['licenses'] = lambda l: licenses(local_globals, l)
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
    return local_globals


# c_argument is magically created for us by pypy.
callbacks = ffi.cast('struct PleaseCallbacks*', c_argument)
callbacks.parse_file = parse_file
callbacks.parse_code = parse_code
callbacks.set_config_value = set_config_value
callbacks.pre_build_callback_runner = run_pre_build_function
callbacks.post_build_callback_runner = run_post_build_function
_add_target = ffi.cast('AddTargetCallback*', callbacks.add_target)
_add_src = ffi.cast('AddStringCallback*', callbacks.add_src)
_add_data = ffi.cast('AddStringCallback*', callbacks.add_data)
_add_dep = ffi.cast('AddStringCallback*', callbacks.add_dep)
_add_exported_dep = ffi.cast('AddStringCallback*', callbacks.add_exported_dep)
_add_tool = ffi.cast('AddStringCallback*', callbacks.add_tool)
_add_out = ffi.cast('AddStringCallback*', callbacks.add_out)
_add_vis = ffi.cast('AddStringCallback*', callbacks.add_vis)
_add_label = ffi.cast('AddStringCallback*', callbacks.add_label)
_add_hash = ffi.cast('AddStringCallback*', callbacks.add_hash)
_add_licence = ffi.cast('AddStringCallback*', callbacks.add_licence)
_add_test_output = ffi.cast('AddStringCallback*', callbacks.add_test_output)
_add_require = ffi.cast('AddStringCallback*', callbacks.add_require)
_add_provide = ffi.cast('AddTwoStringsCallback*', callbacks.add_provide)
_add_named_src = ffi.cast('AddTwoStringsCallback*', callbacks.add_named_src)
_add_command = ffi.cast('AddTwoStringsCallback*', callbacks.add_command)
_set_container_setting = ffi.cast('AddTwoStringsCallback*', callbacks.set_container_setting)
_glob = ffi.cast('GlobCallback*', callbacks.glob)
_get_include_file = ffi.cast('GetIncludeFileCallback*', callbacks.get_include_file)
_get_subinclude_file = ffi.cast('GetIncludeFileCallback*', callbacks.get_subinclude_file)
_get_labels = ffi.cast('GetLabelsCallback*', callbacks.get_labels)
_set_pre_build_callback = ffi.cast('SetBuildFunctionCallback*', callbacks.set_pre_build_function)
_set_post_build_callback = ffi.cast('SetBuildFunctionCallback*', callbacks.set_post_build_function)
_add_dependency = ffi.cast('AddDependencyCallback*', callbacks.add_dependency)
_add_output = ffi.cast('AddOutputCallback*', callbacks.add_output)
_add_licence_post = ffi.cast('AddTwoStringsCallback*', callbacks.add_licence_post)
_set_command = ffi.cast('AddThreeStringsCallback*', callbacks.set_command)
_log = ffi.cast('LogCallback*', callbacks.log)


class ParseError(Exception):
    """Raised on general file parsing errors."""


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
_please_globals['defaultdict'] = defaultdict
_please_globals['ParseError'] = ParseError

# We'll need these guys locally. Unfortunately exec is a statement so we
# can't do it for that.
_compile, _open = compile, open
for k, v in __builtin__.__dict__.items():  # YOLO
    try:
        if issubclass(v, BaseException):
            continue
    except:
        pass
    if k not in _WHITELISTED_BUILTINS:
        del __builtin__.__dict__[k]
