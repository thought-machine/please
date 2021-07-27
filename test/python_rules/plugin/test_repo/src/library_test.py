import unittest
from src import library

class LibraryTest(unittest.TestCase):
    def test_foo(self):
        self.assertEqual("something", library.foo())