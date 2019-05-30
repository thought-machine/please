import unittest


class LocalLibNoImportForYouTest(unittest.TestCase):

    def test_cannot_import_setuptools(self):
        with self.assertRaises(ImportError):
            import setuptools
