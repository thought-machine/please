import unittest


class NumpyTest(unittest.TestCase):

    def test_numpy_is_importable(self):
        import numpy
        self.assertIsNotNone(numpy.nan)
