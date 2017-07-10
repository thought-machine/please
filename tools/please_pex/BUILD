subinclude('//build_defs:go_bindata')

# Extract the contents of __main__.py from pex.
# We could duplicate here easily enough but then might fall out of sync
# with the version of pex we're using.
genrule(
    name = 'main_py',
    srcs = [
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
        '//third_party/python:six',
        'gen_main.sh',
    ],
    outs = ['__main__.py'],
    cmd = '$PKG/gen_main.sh > $OUT',
)

go_bindata(
    name = 'main_files',
    srcs = [
        'pex_main.py',
        'test_main.py',
        ':main_py',
    ],
    package = 'pex',
)

go_library(
    name = 'pex',
    srcs = [
        'pex.go',
        ':main_files',
    ],
    deps = [
        '//third_party/go/zip',
        '//tools/jarcat:zip_writer',
    ],
)

go_binary(
    name = 'pex_main',
    srcs = ['pex_main.go'],
    deps = [
        ':pex',
        '//src/cli',
        '//third_party/go:logging',
    ],
)

# Combine all the Python dependencies into one zipfile and concatenate them
# onto the binary.
genrule(
    name = 'please_pex',
    srcs = [
        ':pex_main',
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
        '//third_party/python:six',
        '//third_party/python:xmlrunner',
        '//third_party/python/py2:coverage',
        '//third_party/python/py3:coverage',
    ],
    outs = ['please_pex'],
    binary = True,
    cmd = ' && '.join([
        # Have to make sure these exist.
        'touch third_party/__init__.py third_party/python/__init__.py',
        'touch tools/__init__.py tools/please_pex/__init__.py',
        'mv third_party/python .bootstrap',
        'mv .bootstrap/pex .bootstrap/_pex',
        'mv .bootstrap/py3/coverage .bootstrap/coverage',
        'mv .bootstrap/py2/coverage/tracer.so .bootstrap/coverage',
        # A little cleanup.
        'rm -rf third_party .bootstrap/xmlrunner/extra .bootstrap/coverage/htmlfiles .bootstrap/py3 .bootstrap/py2 .bootstrap/_pex/bin .bootstrap/.*.pex.zip',
        '$TOOL -d -i .bootstrap -b -o $OUTS --preamble_file $(location :pex_main)',
    ]),
    tools = ['//tools/jarcat'],
    visibility = ['PUBLIC'],
)

python_test(
    name = 'pex_test',
    srcs = ['pex_test.py'],
)

python_test(
    name = 'custom_interpreter_test',
    srcs = ['custom_interpreter_test.py'],
    interpreter = '/usr/bin/pypy',
    labels = ['manual'],
)

python_test(
    name = 'pex_import_test',
    srcs = ['pex_import_test.py'],
    deps = [
        '//third_party/python:dateutil',
        '//third_party/python:requests',
    ],
)
