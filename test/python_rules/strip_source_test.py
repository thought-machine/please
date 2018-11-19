"""Tests on source stripping."""

import os
import subprocess
import unittest
import zipfile


class StripSourceTest(unittest.TestCase):

    def setUp(self):
        self.filename = [x for x in os.environ['DATA'].split(' ') if x.endswith('.pex')][0]

    def test_can_run_binary(self):
        """Test that the dependent binary can be run successfully."""
        subprocess.check_call([self.filename])

    def test_does_not_have_py_file(self):
        """Test that the binary doesn't have the source in it."""
        with zipfile.ZipFile(self.filename) as zf:
            zf.getinfo('test/python_rules/strip_source.pyc')
            with self.assertRaises(KeyError):
                zf.getinfo('test/python_rules/strip_source.py')
