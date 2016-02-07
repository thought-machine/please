genrule(
    name = 'cffi_compile',
    srcs = ['cffi_compiler.py'],
    outs = ['parser_interface.py'],
    cmd = 'pypy $SRC',
)

genrule(
    name = 'embedded_parser',
    srcs = [':cffi_compile', 'please_parser.py'],
    outs = ['embedded_parser.py'],
    cmd = 'cat $SRCS > $OUT',
    visibility = ['//src/parse:all'],
)
