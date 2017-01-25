# Bootstrapping please_pex is difficult, because the code essentially needs to run from a zip
# file. We assemble enough by hand to make it go.
build_rule(
    name = 'please_pex',
    srcs = [
        'pex.py',
        'pex_main.py',
        'test_main.py',
    ],
    outs = ['please_pex'],
    binary = True,
    cmd = ' && '.join([
        # Have to make sure these exist.
        'touch third_party/__init__.py third_party/python/__init__.py',
        'touch tools/__init__.py tools/please_pex/__init__.py',
        'mv third_party/python .bootstrap',
        'mv .bootstrap/pex .bootstrap/_pex',
        # A little cleanup.
        'rm -rf third_party .bootstrap/xmlrunner/extra .bootstrap/coverage/htmlfiles',
        'cat tools/please_pex/pex_main.py | sed -e "s/__MODULE_DIR__//" -e "s|__ENTRY_POINT__|tools.please_pex.pex|" -e "s/__ZIP_SAFE__/True/" > pex_main.py',
        'PYTHONDONTWRITEBYTECODE=1 PYTHONPATH=".:./.bootstrap" python -O tools/please_pex/pex.py --src_dir=${TMP_DIR} --out=${OUT} --entry_point=${PKG//\//.}.pex --bootstrap --interpreter ' + CONFIG.DEFAULT_PYTHON_INTERPRETER,
    ]),
    needs_transitive_deps = True,
    output_is_complete = True,
    deps = [
        '//third_party/python:coverage',
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
        '//third_party/python:six',
        '//third_party/python:xmlrunner',
    ],
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
