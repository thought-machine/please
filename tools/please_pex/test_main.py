"""Customised test runner to output in JUnit-style XML."""

import os
import sys

# This will get templated in by the build rules.
TEST_NAMES = '__TEST_NAMES__'.split(',')



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


def main():
    """Runs the tests. Returns an appropriate exit code."""
    args = [arg for arg in sys.argv[1:] if not arg.startswith('-')]
    # Add .bootstrap dir to path, after the initial pex entry
    sys.path = sys.path[:1] + [os.path.join(sys.path[0], '.bootstrap')] + sys.path[1:]
    if os.getenv('COVERAGE'):
        # It's important that we run coverage while we load the tests otherwise
        # we get no coverage for import statements etc.
        cov = initialise_coverage().coverage()
        cov.start()
        result = run_tests(args)
        cov.stop()
        omissions = ['*/third_party/*', '*/.bootstrap/*', '*/test_main.py']
        # Exclude test code from coverage itself.
        omissions.extend('*/%s.py' % module.replace('.', '/') for module in args)
        import coverage
        try:
            cov.xml_report(outfile=os.getenv('COVERAGE_FILE'), omit=omissions, ignore_errors=True)
        except coverage.CoverageException as err:
            # This isn't fatal; the main time we've seen it is raised for "No data to report" which
            # isn't an exception as far as we're concerned.
            sys.stderr.write('Failed to calculate coverage: %s' % err)
        return result
    else:
        return run_tests(args)
