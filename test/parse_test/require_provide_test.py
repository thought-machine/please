#!/usr/bin/python

import unittest
import zipfile
import __main__ as pex_main


class RequireProvideTest(unittest.TestCase):

    def test_other_language_not_present(self):
        """Test that we don't get the Go file from the dependent rule."""
        with zipfile.ZipFile(pex_main.PEX) as zf:
            with self.assertRaises(KeyError):
                zf.getinfo('test/parse_test/test_require.go')

    def test_our_language_is_present(self):
        """Test that we do get the Python file from the dependent rule."""
        with zipfile.ZipFile(pex_main.PEX) as zf:
            zf.getinfo('test/parse_test/test_require.py')


if __name__ == '__main__':
    unittest.main()
