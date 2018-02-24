import sys
import unittest


class ConfigTest(unittest.TestCase):

    def test_flag_matches(self):
        """Test the flag matches as expected. It's always x86_64 because we don't build Please for x86."""
        self.assertEqual('--arch=x86_64', sys.argv[-1])
