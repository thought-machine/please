"""Tests on source stripping."""

import pkg_resources
import unittest


class PexTest(unittest.TestCase):

    def testFileIsAPyc(self):
        """Test that we are running from a .pyc."""
        self.assertTrue(__file__.endswith('.pyc'))

    def testThisFileDoesntExist(self):
        """Test this file doesn't exist in the pex."""
        self.assertFalse(pkg_resources.resource_exists('src.build.python', 'strip_source_test.py'))
