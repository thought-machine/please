import unittest


class TestProtoLibrary(unittest.TestCase):

    def test_can_import_library(self):
        """Test that we can import the library.

        The real test is at build time, this is really just here to make
        sure that it gets built, and we might as well verify things are OK.
        """
        from test.python_rules import test_pb2
