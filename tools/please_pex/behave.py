import os


def get_features_dir():
    for i in TEST_NAMES:
        file_name = i.split('/')[-1]
        if '.feature' in file_name:
            return os.path.dirname(i)


def run_tests(args=None):
    from behave.__main__ import main

    args = args or []
    features_dir = get_features_dir()
    args += [features_dir, '--junit',
             '--junit-directory', 'test.results']
    main(args)
