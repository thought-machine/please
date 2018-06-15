import os


def run_tests(args=None):
    from behave.__main__ import main
    default_args = [os.environ.get('PKG_DIR'), '--junit',
                    '--junit-directory', 'test.results']
    if args:
        args += default_args
    else:
        args = default_args
    main(args)
