ENGINES = {
    'pypy': [],
    'python2': ['//third_party/python/py2:cffi'],
    'python3': ['//third_party/python/py3:cffi'],
}

for interpreter, deps in ENGINES.items():
    extension = 'dylib' if CONFIG.OS == 'darwin' else 'so'
    genrule(
        name = 'please_parser_' + interpreter,
        srcs = ['cffi_compiler.py', 'defs.h', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (interpreter, extension)],
        deps = ['//third_party/python:pycparser'] + deps,
        cmd = 'export PYTHONPATH="third_party/python:third_party/python/py2:third_party/python/py3"; $TOOL $SRCS --verbose',
        visibility = ['PUBLIC'],
        # Have to make sure we get the right interpreter so we can use our cffi.
        # TODO(pebers): Should really have a more robust way of handling all of this.
        tools = ['python3.5' if interpreter == 'python3' else interpreter],
    )

filegroup(
    name = 'all_engines',
    srcs = [':please_parser_' + interpreter for interpreter in ENGINES],
    visibility = ['PUBLIC'],
)
