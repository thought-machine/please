"""Simple test that our pexer works correctly for a unit test."""

import unittest


class PexTest(unittest.TestCase):

    def testSuccess(self):
        """Test records a success."""
        self.assertEqual(4, 2**2)

    @unittest.skip("Saving this for when the stars are right")
    def testSkipped(self):
        """Test skipping a test"""
        self.assertEqual(10, 3**3)

    @unittest.expectedFailure
    def testSkipped(self):
        """Test an expected failure"""
        self.assertEqual(132, 4**4)


if __name__ == '__main__':
    unittest.main()
