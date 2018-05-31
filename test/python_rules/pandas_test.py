import unittest


class PandasTest(unittest.TestCase):

    def test_import(self):
        import pandas as pd
        pd.DataFrame()

    def test_third_party_import(self):
        from third_party.python import pandas as pd2
        import pandas as pd
        df = pd.DataFrame()
        self.assertIsInstance(df, pd2.DataFrame)
