python_binary(
    name = 'please_pex',
    main = 'pex.py',
    deps = [
        ':main_files',
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
    ],
    visibility = ['PUBLIC'],
)

python_library(
    name = 'main_files',
    srcs = ['test_main.py', 'pex_main.py'],
    deps = [
        '//third_party/python:six',
        '//third_party/python:xmlrunner',
        '//third_party/python:coverage',
    ],
)

python_test(
    name = 'pex_test',
    srcs = ['pex_test.py'],
)

python_test(
    name = 'custom_interpreter_test',
    srcs = ['custom_interpreter_test.py'],
    labels = ['manual'],
    interpreter = '/usr/bin/pypy'
)

# This is needed for bootstrapping the build system: we have to be
# able to build a .pex with the pex builder which is itself a pex.
build_rule(
    name = 'bootstrap_pexer',
    srcs = ['pex.py'],
    outs = ['bootstrap_pexer.pex'],
    deps = [
        ':main_files',
        '//third_party/python:pex',
        '//third_party/python:pkg_resources',
    ],
    cmd = ' && '.join([
        # Have to make sure these exist.
        'touch third_party/__init__.py third_party/python/__init__.py',
        'touch src/__init__.py src/build/__init__.py src/build/python/__init__.py',
        'PYTHONPATH="." python src/build/python/pex.py --src_dir=${TMP_DIR} --out=${OUT} --entry_point=${PKG//\//.}.pex --interpreter ' + CONFIG.DEFAULT_PYTHON_INTERPRETER,
    ]),
    needs_transitive_deps = True,
    binary = True,
    output_is_complete = True,
    visibility = ['PUBLIC']
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
    deps = [
        '//src/build/cc:so_test_py'
    ],
    labels = ['cc'],  # Depends on cc rules to build
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
    deps = [':data_dep'],
    data = [':data_dep'],
)
