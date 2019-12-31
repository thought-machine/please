import os
import sys
import unittest
from importlib import import_module


def list_classes(suite):
    for test in suite:
        if isinstance(test, unittest.suite.TestSuite):
            for cls, name in list_classes(test):
                yield cls, name
        else:
            yield test, test.__class__.__module__ + '.' + test.id()


def get_suite(test_names, raise_on_empty=False):
    """
    Get the test suite to run the test

    :param test_names: Name of the tests to be filtered
    :param raise_on_empty: raise Exception if result filtered is empty

    :return: unittest.suite.TestSuite
    """
    suite = unittest.TestSuite(unittest.defaultTestLoader.loadTestsFromModule(module)
                               for module in import_tests())
    # Filter to test name only, this ensures the extra flags does not get swallowed
    test_names = list(filter(lambda x: not x.startswith('-'), test_names))

    # filter results if test_names is not empty
    if test_names:
        new_suite = unittest.suite.TestSuite()
        for name in test_names:
            new_suite.addTests(cls for cls, class_name in list_classes(suite)
                               if name in class_name)
        if raise_on_empty and suite.countTestCases() == 0:
            raise Exception('No matching tests found')

        return new_suite

    return suite


def import_tests():
    """Yields the set of test modules, from file if necessary."""
    # We have files available locally, but there may (likely) also be python files in the same
    # Python package within the pex. We can't just import them because the parent package exists
    # in only one of those places (this is similar to importing generated code from plz-out/gen).
    for filename in TEST_NAMES:
        pkg_name, _ = os.path.splitext(filename.replace('/', '.'))
        try:
            yield import_module(pkg_name)
        except ImportError:
            with open(filename, 'r') as f:
                if PY_VERSION.major < 3:
                    mod = imp.load_module(pkg_name, f, filename, ('.py', 'r', imp.PY_SOURCE))
                else:
                    mod = machinery.SourceFileLoader(pkg_name, filename).load_module()

                # Have to set the attribute on the parent module too otherwise some things
                # can't find it.
                parent, _, mod_name = pkg_name.rpartition('.')
                if parent and parent in sys.modules:
                    setattr(sys.modules[parent], mod_name, mod)
                yield mod


def run_tests(test_names):
    """Runs tests using unittest, returns the number of failures."""
    # N.B. import must be deferred until we have set up import paths.
    import xmlrunner
    suite = get_suite(test_names, raise_on_empty=True)

    runner = xmlrunner.XMLTestRunner(output='test.results', outsuffix='')
    results = runner.run(suite)
    return len(results.errors) + len(results.failures)
