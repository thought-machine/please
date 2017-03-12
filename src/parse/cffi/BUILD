ENGINES = [
    'pypy',
    'python2',
    'python3',
]

for interpreter in ENGINES:
    extension = 'dylib' if CONFIG.OS == 'darwin' else 'so'
    genrule(
        name = 'please_parser_' + interpreter,
        srcs = ['cffi_compiler.py', 'defs.h', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (interpreter, extension)],
        cmd = interpreter + ' $SRCS --verbose',
        visibility = ['PUBLIC'],
    )

filegroup(
    name = 'all_engines',
    srcs = [':please_parser_' + interpreter for interpreter in ENGINES],
    visibility = ['PUBLIC'],
)
