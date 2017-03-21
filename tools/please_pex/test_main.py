"""Customised test runner to output in JUnit-style XML."""

import argparse
import os
import unittest
import sys
try:
    from builtins import __import__  # python3
except ImportError:
    from __builtin__ import __import__  # python2

# This will get templated in by the build rules.
TEST_NAMES = '__TEST_NAMES__'.split(',')

sys.path.append(os.path.join(sys.argv[0], '.bootstrap'))

import xmlrunner
from pex_main import clean_sys_path, override_import


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


def initialise_coverage():
    """Imports & initialises the coverage module."""
    sys.meta_path.append(TracerImport())
    import coverage
    from coverage import control as coverage_control
    _original_xml_file = coverage_control.XmlReporter.xml_file
    # Fix up paths in coverage output which are absolute; we want paths relative to
    # the repository root. Also skip empty __init__.py files.
    def _xml_file(self, fr, analysis):
        if '.pex' in fr.filename:
            fr.filename = fr.filename[fr.filename.index('.pex') + 5:]  # +5 to take off .pex/
        if not (fr.filename.endswith('__init__.py') and len(analysis.statements) <= 1):
            analysis.filename = fr.filename
            fr.relname = fr.filename
            _original_xml_file(self, fr, analysis)
    coverage_control.XmlReporter.xml_file = _xml_file
    return coverage


class TracerImport(object):
    """Import hook to load coverage.tracer.

    This module is binary and improves the speed of coverage quite a bit, but of course isn't
    easy for us to get out of a .pex file. We also have to package multiple forms of it.
    """

    def find_module(self, fullname, path=None):
        if fullname == 'coverage.tracer':
            return self  # we only handle this one module

    def load_module(self, fullname):
        mod = sys.modules.get(fullname)
        if mod:
            return mod
        import imp
        import zipfile
        try:
            zf = zipfile.ZipFile(sys.argv[0])
        except Exception:
            raise ImportError('Failed to open pex, coverage.tracer will not be available\n')
        for suffix, mode, type in imp.get_suffixes():
            try:
                filename = zf.extract(os.path.join('.bootstrap/coverage', 'tracer' + suffix), '.')
                with open(filename, mode) as f:
                    return imp.load_module(fullname, f, filename, (suffix, mode, type))
            except Exception:
                pass  # We don't expect to extract any individual suffix.
        raise ImportError('Failed to extract coverage.tracer module, coverage will be slower\n')


def run_tests(test_names):
    """Runs tests, returns the number of failures."""
    # unittest's discovery produces very misleading errors in some cases; if it tries to import
    # a module which imports other things that then fail, it reports 'module object has no
    # attribute <test name>' and swallows the original exception. Try to import them all first
    # so we get better error messages.
    for test_name in TEST_NAMES:
        __import__(test_name)
    suite = unittest.defaultTestLoader.loadTestsFromNames(TEST_NAMES)
    if test_names:
        suite = filter_suite(suite, test_names)
        if suite.countTestCases() == 0:
            raise Exception('No matching tests found')
    runner = xmlrunner.XMLTestRunner(output='test.results', outsuffix='')
    results = runner.run(suite)
    return len(results.errors) + len(results.failures)


def main(args, extra_args):
    args.test_names = args.test_names.split(',') if args.test_names else []
    sys.argv[1:] = extra_args
    if args.list_classes:
        suite = unittest.defaultTestLoader.loadTestsFromNames(TEST_NAMES)
        for _, cls in set(list_classes(suite)):
            sys.stdout.write(cls + '\n')
        return 0
    elif args.coverage or os.getenv('COVERAGE'):
        # It's important that we run coverage while we load the tests otherwise
        # we get no coverage for import statements etc.
        cov = initialise_coverage().coverage()
        cov.start()
        result = run_tests(args.test_names)
        cov.stop()
        omissions = ['*/third_party/*', '*/.bootstrap/*', '*/test_main.py']
        # Exclude test code from coverage itself.
        omissions.extend('*/%s.py' % module.replace('.', '/') for module in args.test_names)
        cov.xml_report(outfile=os.getenv('COVERAGE_FILE'), omit=omissions, ignore_errors=True)
        return result
    else:
        return run_tests(args.test_names)


if __name__ == '__main__':
    override_import()
    clean_sys_path()
    parser = argparse.ArgumentParser(description='Arguments for Please Python tests.')
    parser.add_argument('--list_classes', type=bool, default=False, help='List all test classes')
    parser.add_argument('--coverage', dest='coverage', action='store_true',
                        help='Write output coverage file')
    parser.add_argument('--test_names', default=os.environ.get('TEST_ARGS'), help='Tests to run')
    sys.exit(main(*parser.parse_known_args()))
