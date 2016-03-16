import random
import unittest


class FlakyTest(unittest.TestCase):

    def test_flaky(self):
        self.assertEqual(1, random.choice([1, 2, 3]))
