python_binary(
    name = 'please_pex',
    main = 'pex.py',
    visibility = ['PUBLIC'],
    deps = [
        ':main_files',
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
    ],
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
        'touch src/__init__.py src/build/__init__.py src/build/python/__init__.py',
        'PYTHONPATH="." python -O src/build/python/pex.py --src_dir=${TMP_DIR} --out=${OUT} --entry_point=${PKG//\//.}.pex --interpreter ' + CONFIG.DEFAULT_PYTHON_INTERPRETER,
    ]),
    needs_transitive_deps = True,
    output_is_complete = True,
    visibility = ['PUBLIC'],
    deps = [
        # Depending on these private targets is not nice, but works around
        # a circular dependency because this tool is used to precompile the library.
        '//third_party/python:_pex#install',
        '//third_party/python:_pkg_resources#install',
        '//third_party/python:_six#install',
        '//third_party/python:_xmlrunner#install',
        '//third_party/python:_coverage#install',
    ],
)

python_test(
    name = 'pex_import_test',
    srcs = ['pex_import_test.py'],
    deps = [
        '//third_party/python:dateutil',
        '//third_party/python:requests',
    ],
)

# Used to test zip-safety flags.
python_test(
    name = 'zip_unsafe_test',
    srcs = ['zip_unsafe_test.py'],
    labels = ['cc'],  # Depends on cc rules to build
    deps = [
        '//src/build/cc:so_test_py',
    ],
)

# Tests for a subtle case where python_test rules should get a pex when
# specified as data, but not when depending directly.
python_binary(
    name = 'data_dep',
    main = 'data_dep.py',
)

python_test(
    name = 'data_dep_test',
    srcs = ['data_dep_test.py'],
    data = [':data_dep'],
    deps = [':data_dep'],
)
