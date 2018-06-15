import os


def run_tests(args=None):
    from behave.__main__ import main
    args = args or []

    args += [os.environ.get('PKG_DIR'), '--junit',
             '--junit-directory', 'test.results']
    main(args)
