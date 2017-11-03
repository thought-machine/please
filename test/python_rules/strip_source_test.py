"""Tests on source stripping."""

import unittest
import zipfile


class StripSourceTest(unittest.TestCase):

    def test_file_is_a_pyc(self):
        """Test that the stripped module is a .pyc."""
        from test.python_rules import strip_source
        self.assertTrue(strip_source.__file__.endswith('.pyc'))

    def test_this_file_doesnt_exist(self):
        """Test this file doesn't exist in the pex."""
        import __main__ as pex_main
        with zipfile.ZipFile(pex_main.PEX, 'r') as zf:
            with self.assertRaises(KeyError):
                zf.getinfo('test/python_rules/strip_source.py')
