import random
import unittest


class FlakinessTest(unittest.TestCase):

    def test_flakiness(self):
        """This test is deliberately flaky to test that functionality."""
        self.assertLess(random.random(), 0.3)


if __name__ == '__main__':
    unittest.main()
