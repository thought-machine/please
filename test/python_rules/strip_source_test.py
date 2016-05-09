"""Tests on source stripping."""

import pkg_resources
import unittest


class StripSourceTest(unittest.TestCase):

    def test_file_is_a_pyc(self):
        """Test that we are running from a .pyc."""
        self.assertTrue(__file__.endswith('.pyc'))

    def test_this_file_doesnt_exist(self):
        """Test this file doesn't exist in the pex."""
        self.assertFalse(pkg_resources.resource_exists('test.python_rules', 'strip_source_test.py'))
