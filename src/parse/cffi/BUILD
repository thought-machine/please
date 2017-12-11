subinclude('//build_defs:engines')

# Dynamic interpreter engines. These can be specified at runtime (using -o parse.engine)
# and something similar is used at bootstrap time.
# For now we keep them around for reference but they aren't shipped.
for interpreter, deps in ENGINE_DEPENDENCIES.items():
    extension = 'dylib' if CONFIG.OS == 'darwin' else 'so'
    genrule(
        name = 'please_parser_' + interpreter,
        srcs = ['cffi_compiler.py', 'defs.h', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (interpreter, extension)],
        deps = ['//third_party/python:pycparser'] + deps,
        cmd = 'export PYTHONPATH="third_party/python:third_party/python/cffi"; $TOOL $SRCS --verbose',
        tools = [interpreter],
        labels = ['manual'],
    )

genrule(
    name = 'parser_src',
    srcs = [
        'cffi_compiler.py',
        'defs.h',
        'please_parser.py',
    ],
    outs = ['parser_interface.c'],
    cmd = 'export PYTHONPATH="third_party/python:third_party/python/cffi"; $TOOL $SRCS --verbose',
    tools = [PARSER_TOOL],
    deps = [
        '//third_party/python:cffi',
        '//third_party/python:pycparser',
    ],
)

c_static_library(
    name = 'please_parser_static',
    srcs = [':parser_src'],
    hdrs = ['defs.h'],
    pkg_config_libs = [PARSER_PKG_CONFIG],
    visibility = ['//src/parse/...'],
)
