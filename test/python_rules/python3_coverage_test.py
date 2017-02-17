import unittest


class Python3CoverageTest(unittest.TestCase):
    """Tests that we can import parts of the coverage module.

    By default it's not trivial to import binary modules from inside a pex. Also we need to package
    relevant versions of the tracer module, which can't work for every OS, but it'd be nice to
    cover at least a few since it improves performance quite a bit.
    """

    @classmethod
    def setUpClass(cls):
        """Must ensure coverage is set up before we can import tracer."""
        import test_main
        test_main.initialise_coverage()

    def test_can_import_coverage(self):
        """Test we can import the coverage module OK."""
        import coverage
        self.assertIsNotNone(coverage)

    def test_can_import_tracer(self):
        """Test we can import the binary tracer module."""
        from coverage import tracer
        self.assertIsNotNone(tracer)
