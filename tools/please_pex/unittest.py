import unittest
from importlib import import_module


def list_classes(suite):
    for test in suite:
        if isinstance(test, unittest.suite.TestSuite):
            for cls, name in list_classes(test):
                yield cls, name
        else:
            yield test, test.__class__.__module__ + '.' + test.id()


def filter_suite(suite, test_names):
    """Reduces a test suite to just the tests matching the given names."""
    new_suite = unittest.suite.TestSuite()
    for name in test_names:
        new_suite.addTests(cls for cls, class_name in list_classes(suite) if name in class_name)
    return new_suite


def run_tests(test_names):
    """Runs tests using unittest, returns the number of failures."""
    # N.B. import must be deferred until we have set up import paths.
    import xmlrunner
    # unittest's discovery produces very misleading errors in some cases; if it tries to import
    # a module which imports other things that then fail, it reports 'module object has no
    # attribute <test name>' and swallows the original exception. Try to import them all first
    # so we get better error messages.
    for test_name in TEST_NAMES:
        import_module(test_name)
    suite = unittest.defaultTestLoader.loadTestsFromNames(TEST_NAMES)
    if test_names:
        suite = filter_suite(suite, test_names)
        if suite.countTestCases() == 0:
            raise Exception('No matching tests found')
    runner = xmlrunner.XMLTestRunner(output='test.results', outsuffix='')
    results = runner.run(suite)
    return len(results.errors) + len(results.failures)
