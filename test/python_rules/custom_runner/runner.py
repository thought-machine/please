import os
import pytest


def run(test_names, args):
    """Custom test runner entry point.

    This is fairly minimal and serves mostly to demonstrate how to define this as an
    entry point.

    Args:
      test_names: The names of the original test modules to be run (i.e. the things that were
                  srcs to the python_test rule).
      args: Any command-line arguments, not including sys.argv[0].
    """
    results_file = os.getenv('RESULTS_FILE', 'test.results')
    os.mkdir(results_file)
    args += ['--junitxml', os.path.join(results_file, 'results.xml')] + test_names
    if os.environ.get('DEBUG'):
        args.append('--pdb')
    return pytest.main(args)
