filegroup(
    name = 'pex_data',
    srcs = [
        'pex_main.py',
        'pex_run.py',
        'pytest.py',
        'test_main.py',
        'unittest.py',
    ],
    visibility = ['//tools/please_pex/pex:all'],
)

go_binary(
    name = 'pex_main',
    srcs = ['pex_main.go'],
    deps = [
        '//src/cli',
        '//third_party/go:logging',
        '//tools/please_pex/pex',
    ],
)

# Combine all the Python dependencies into one zipfile and concatenate them
# onto the binary.
genrule(
    name = 'please_pex',
    srcs = [
        ':pex_main',
        '//third_party/python:six',
        '//third_party/python:xmlrunner',
        '//third_party/python:coverage',
        '//third_party/python:pytest',
        '//third_party/python:py',
        '//third_party/python:pluggy',
        '//third_party/python:attrs',
        '//third_party/python:funcsigs',
        '//third_party/python:pkg_resources',
    ],
    outs = ['please_pex'],
    binary = True,
    cmd = [
        # Have to make sure these exist.
        'touch third_party/__init__.py third_party/python/__init__.py',
        'touch tools/__init__.py tools/please_pex/__init__.py',
        'mv third_party/python .bootstrap',
        # A little cleanup.
        'rm -rf third_party .bootstrap/xmlrunner/extra .bootstrap/coverage/htmlfiles .bootstrap/.*.pex.zip',
        '$TOOL z -d -i .bootstrap -o $OUTS --preamble_file $(location :pex_main)',
    ],
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
