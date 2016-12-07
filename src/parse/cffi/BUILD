ENGINES = [
    'pypy',
    'python2',
    'python3',
]

def cffi_binary(interpreter, tag=None):
    return build_rule(
        name = 'please_parser_' + interpreter,
        tag = tag,
        srcs = ['cffi_compiler.py', 'defs.h', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (interpreter, extension)],
        cmd = interpreter + ' $SRCS --verbose',
        visibility = ['PUBLIC'],
    )

for interpreter in ENGINES:
    extension = 'dylib' if CONFIG.OS == 'darwin' else 'so'
    cmd = interpreter + ' $SRCS --verbose'
    if CONFIG.OS == 'linux' and interpreter == 'pypy':
        # Bit of an exciting hack to support dynamically downloading a copy of PyPy.
        cmd += ' && $TOOL --set-rpath ./plz-out/gen/_remote/_pypy/bin $OUT'
    genrule(
        name = 'please_parser_' + interpreter,
        srcs = ['cffi_compiler.py', 'defs.h', 'please_parser.py'],
        outs = ['libplease_parser_%s.%s' % (interpreter, extension)],
        tools = [':patchelf'] if CONFIG.OS == 'linux' else None,
        cmd = cmd,
        visibility = ['PUBLIC'],
    )

# Temporarily removed so it doesn't wind up in the deployment; it's not ready yet.
ENGINES.remove('python3')

filegroup(
    name = 'all_engines',
    srcs = [':please_parser_' + interpreter for interpreter in ENGINES],
    visibility = ['PUBLIC'],
)

remote_file(
    name = 'patchelf',
    out = 'patchelf',
    binary = True,
    hashes = ['fab293cec73c91d4350b6f8387ccc55400bb5dd3'],
    url = 'https://get.please.build/linux_amd64/tools/patchelf-0.9',
)
