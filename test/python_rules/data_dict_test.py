import os
import unittest


class DataDictTest(unittest.TestCase):

    def test_load_file(self):
        with open(os.environ.get('DATA_TXT')) as f:
            expected = 'how much wood could a woodchuck chuck if a woodchuck could chuck wood?'
            self.assertEqual(expected, f.read().strip())
