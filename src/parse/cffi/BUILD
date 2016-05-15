for engine in ['pypy', 'python2', 'python3']:
    extension = 'dylib' if CONFIG.OS == 'darwin' else 'so'
    genrule(
        name = 'please_parser_' + engine,
        srcs = ['cffi_compiler.py', '//src/parse:defs_hdr', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (engine, extension)],
        cmd = engine + ' $SRCS --verbose',
        visibility = ['PUBLIC'],
    )
