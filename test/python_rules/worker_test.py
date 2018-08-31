"""
Testing the integration with a worker script
"""
import unittest


class WorkerTest(unittest.TestCase):
    def test_dummy(self):
        self.assertEqual(11, 11)
