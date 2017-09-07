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

filegroup(
    name = 'all_engines',
    srcs = [':please_parser_' + interpreter for interpreter in ENGINES],
    visibility = ['PUBLIC'],
)
