"""Test that data for a test isn't also packed into the .pex."""

import os
import pkg_resources
import subprocess
import unittest


class DataDepTest(unittest.TestCase):

    def test_cannot_import_data_dep(self):
        """Test that the file didn't get packed into the pex."""
        self.assertFalse(pkg_resources.resource_exists('test.python_rules', 'data_dep.pex'))
        self.assertFalse(pkg_resources.resource_exists('test.python_rules', 'data_dep.py'))

    def test_can_run_data_dep(self):
        """Test that we can also invoke the .pex directly as a data dependency."""
        output = subprocess.check_output(['test/python_rules/data_dep.pex'])
        self.assertEqual('42', output.strip().decode('utf-8'))
