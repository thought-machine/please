subinclude('//build_defs:engines')

for interpreter, deps in ENGINE_DEPENDENCIES.items():
    extension = 'dylib' if CONFIG.OS == 'darwin' else 'so'
    genrule(
        name = 'please_parser_' + interpreter,
        srcs = ['cffi_compiler.py', 'defs.h', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (interpreter, extension)],
        deps = ['//third_party/python:pycparser'] + deps,
        cmd = 'export PYTHONPATH="third_party/python:third_party/python/cffi"; $TOOL $SRCS --verbose',
        visibility = ['PUBLIC'],
        tools = [interpreter],
    )

# Experimental static interpreter.
genrule(
    name = 'parser_src',
    srcs = [
        'cffi_compiler.py',
        'defs.h',
        'please_parser.py',
    ],
    outs = ['parser_interface.c'],
    cmd = 'export PYTHONPATH="third_party/python:third_party/python/cffi"; $TOOL $SRCS --verbose',
    tools = ['python2'],
    deps = ['//third_party/python:pycparser'] + deps,
)

c_static_library(
    name = 'please_parser_static',
    srcs = [':parser_src'],
    hdrs = ['defs.h'],
    pkg_config_libs = [PARSER_PKG_CONFIG],
    deps = [':libpython'],
    visibility = ['//src/parse/...'],
)

# This rule locates libpython which we need the symbols from.
# This is pretty ugly but right now I can't persuade it to link statically for some reason
# (always end up getting a dynamic link instead).
# It's also pretty yuck that it arbitrarily fetches this off the system, although
# arguably it is not really different to using pkg-config in that regard.
genrule(
    name = 'libpython',
    srcs = ['find_python.sh'],
    outs = [PARSER_LIB_FILENAME],
    cmd = 'bash $SRCS %s %s' % (PARSER_PKG_CONFIG, PARSER_LIB_FILENAME),
)

filegroup(
    name = 'all_engines',
    srcs = [':please_parser_' + interpreter for interpreter in ENGINES],
    visibility = ['PUBLIC'],
)
