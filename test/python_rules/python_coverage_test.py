import sys
import unittest


class PythonCoverageTest(unittest.TestCase):
    """Tests that we can import parts of the coverage module.

    By default it's not trivial to import binary modules from inside a pex. Also we need to package
    relevant versions of the tracer module, which can't work for every OS, but it'd be nice to
    cover at least a few since it improves performance quite a bit.
    """

    @classmethod
    def setUpClass(cls):
        """Must ensure coverage is set up before we can import tracer."""
        import __main__ as test_main
        test_main.initialise_coverage()

    def test_can_import_coverage(self):
        """Test we can import the coverage module OK."""
        import coverage
        self.assertIsNotNone(coverage)

    @unittest.skipIf(sys.platform == 'darwin' and sys.version_info.major < 3,
                     'Not working on OSX python2 at present due to symbol errors')
    def test_can_import_tracer(self):
        """Test we can import the binary tracer module."""
        from coverage import tracer
        self.assertIsNotNone(tracer)

    def test_coverage_output(self):
        """Test for manually examining coverage output."""
        from test.python_rules.python_coverage import the_answer
        self.assertEqual(42, the_answer())
