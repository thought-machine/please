import unittest


class SciPyTest(unittest.TestCase):

    def test_can_import(self):
        # This fails to import if we put an __init__.py in the wrong place.
        from third_party.python.scipy import stats
