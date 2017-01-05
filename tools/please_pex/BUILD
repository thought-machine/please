python_binary(
    name = 'please_pex',
    out = 'please_pex',
    main = 'pex.py',
    shebang = '/usr/bin/env python',
    deps = [
        ':main_files',
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
    ],
    visibility = ['PUBLIC'],
)

python_library(
    name = 'main_files',
    srcs = [
        'pex_main.py',
        'test_main.py',
    ],
    deps = [
        '//third_party/python:coverage',
        '//third_party/python:six',
        '//third_party/python:xmlrunner',
    ],
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

# This is needed for bootstrapping the build system: we have to be
# able to build a .pex with the pex builder which is itself a pex.
build_rule(
    name = 'bootstrap_pexer',
    srcs = [
        'pex.py',
        'pex_main.py',
        'test_main.py',
    ],
    outs = ['bootstrap_pexer.pex'],
    binary = True,
    cmd = ' && '.join([
        # Have to make sure these exist.
        'touch third_party/__init__.py third_party/python/__init__.py',
        'touch tools/__init__.py tools/please_pex/__init__.py',
        'PYTHONPATH="." python -O tools/please_pex/pex.py --src_dir=${TMP_DIR} --out=${OUT} --entry_point=${PKG//\//.}.pex --interpreter ' + CONFIG.DEFAULT_PYTHON_INTERPRETER,
    ]),
    needs_transitive_deps = True,
    output_is_complete = True,
    deps = [
        # Depending on these private targets is not nice, but works around
        # a circular dependency because this tool is used to precompile the library.
        '//third_party/python:_pex#install',
        '//third_party/python:_pkg_resources#install',
        '//third_party/python:_six#install',
        '//third_party/python:_xmlrunner#install',
        '//third_party/python:_coverage#install',
    ],
    visibility = ['PUBLIC'],
)

python_test(
    name = 'pex_import_test',
    srcs = ['pex_import_test.py'],
    deps = [
        '//third_party/python:dateutil',
        '//third_party/python:requests',
    ],
)
