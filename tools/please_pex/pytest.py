import os


def run_tests(args):
    """Runs tests using pytest, returns the number of failures."""
    # N.B. import must be deferred until we have set up import paths.
    from pytest import main
    args = args or []

    # Added this so the original filter functionality by name does not change
    if args and not args[0].startswith('-'):
        args += ['-k', ' '.join(args)]

    args += ['--junitxml', 'test.results'] + TEST_NAMES

    if os.environ.get('DEBUG'):
        args.append('--pdb')
    return main(args)
