import unittest


class PandasTest(unittest.TestCase):

    def test_import(self):
        import pandas as pd
        pd.DataFrame()
