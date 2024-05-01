import sys
import unittest


class ConfigTest(unittest.TestCase):

    def test_flag_matches(self):
        """Test the flag matches as expected."""
        self.assertEqual('--word_size=64', sys.argv[-1])
