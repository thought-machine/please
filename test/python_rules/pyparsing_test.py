import unittest


class PyparsingTest(unittest.TestCase):

    def test_pyparsing_is_importable(self):
        import pyparsing
        self.assertIsNotNone(pyparsing.alphas)
