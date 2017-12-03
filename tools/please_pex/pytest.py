def run_tests(test_names):
    """Runs tests using pytest, returns the number of failures."""
    # N.B. import must be deferred until we have set up import paths.
    from pytest import main
    args = ['--junitxml', 'test.results'] + TEST_NAMES
    if test_names:
        args += ['-k', ' '.join(test_names)]
    return main(args)
