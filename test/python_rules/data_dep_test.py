"""Test on deps, data, requires and provides."""

import os
import subprocess
import unittest


class DataDepTest(unittest.TestCase):

    def test_direct_dep(self):
        """Test that we can import the module directly."""
        from test.python_rules import data_dep
        self.assertEqual(42, data_dep.the_answer())

    def test_data_dep(self):
        """Test that we can also invoke the .pex directly as a data dependency."""
        output = subprocess.check_output(['test/python_rules/data_dep.pex'])
        self.assertEqual('42', output.strip().decode('utf-8'))


# Deliberately not adding an "if __name__ == '__main__':" stanza since this test
# will only work when invoked through plz so the data dep is in the right place.
