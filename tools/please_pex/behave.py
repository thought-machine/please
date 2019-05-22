import os
import logging
from pathlib import Path


def get_features_dir():
    for i in TEST_NAMES:
        file_name = i.split('/')[-1]
        if '.feature' in file_name:
            return [os.path.dirname(i)]


def get_all_feature_files():
    return [str(feature) for feature in Path('.').glob('**/*.feature')]


def run_tests(args=None):
    from behave.__main__ import main

    args = args or []

    if 'all_features=True' in args:
        logging.info('Getting all feature files')
        features = get_all_feature_files()
    else:
        logging.info('Getting a single feature dir')
        features = get_features_dir()

    args += features + [ '--junit', '--junit-directory', 'test.results']
    main(args)
