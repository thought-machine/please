#!/usr/bin/python

import pkg_resources
import unittest


class RequireProvideTest(unittest.TestCase):

    def test_other_language_not_present(self):
        """Test that we don't get the Go file from the dependent rule."""
        self.assertFalse(pkg_resources.resource_exists('src.parse', 'test_require.go'))

    def test_our_language_is_present(self):
        """Test that we do get the Python file from the dependent rule."""
        self.assertTrue(pkg_resources.resource_exists('src.parse', 'test_require.py'))


if __name__ == '__main__':
    unittest.main()
