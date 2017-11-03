"""Test that data for a test isn't also packed into the .pex."""

import os
import subprocess
import unittest
import zipfile


class DataDepTest(unittest.TestCase):

    def test_cannot_import_data_dep(self):
        """Test that the file didn't get packed into the pex."""
        import __main__ as pex_main
        with zipfile.ZipFile(pex_main.PEX) as zf:
            with self.assertRaises(KeyError):
                zf.getinfo('test/python_rules/data_dep.pex')
            with self.assertRaises(KeyError):
                zf.getinfo('test/python_rules/data_dep.py')

    def test_can_run_data_dep(self):
        """Test that we can also invoke the .pex directly as a data dependency."""
        output = subprocess.check_output(['test/python_rules/data_dep.pex'])
        self.assertEqual('42', output.strip().decode('utf-8'))
