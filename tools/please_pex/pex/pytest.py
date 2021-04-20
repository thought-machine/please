import os


def run_tests(args):
    """Runs tests using pytest, returns the number of failures."""
    # N.B. import must be deferred until we have set up import paths.
    from pytest import main
    args = args or []

    # Added this so the original filter functionality by name does not change
    if args:
        filtered_tests = ''

        for i in args[:]:
            if not i.startswith('-'):
                filtered_tests += i + ' '
                args.remove(i)

        if filtered_tests:
            try:
                args.remove('-k')
            except ValueError:
                pass
            args += ['-k', filtered_tests.strip()]

    # It's easier if all python_test rules output into a directory.
    results_file = os.getenv('RESULTS_FILE', 'test.results')
    os.mkdir(results_file)
    args += ['--junitxml', os.path.join(results_file, 'results.xml')] + TEST_NAMES

    if os.environ.get('DEBUG'):
        args.append('--pdb')

    return main(args)
