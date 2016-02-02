"""Library to test transitive zip_safe flag."""

import unittest

from src.build.cc import so_test


class SharedObjectTest(unittest.TestCase):

    def test_file1_contents(self):
        contents = so_test.get_embedded_file_1()
        self.assertEqual('testing message 1\n', contents)


if __name__ == '__main__':
    unittest.main()
