"""What a .pex has to get right on Windows, none of which is visible on Linux.

The interesting part is not the assertions - it is that this runs at all. Getting here means
please_pex prepended a preamble Windows will execute, that preamble found a Python interpreter
and handed it the .pex, and Python imported this module out of the zip.
"""

import os
import sys
import unittest


class PexTest(unittest.TestCase):
    def test_running_on_windows(self):
        """The whole point: this is a Windows interpreter, not the host one."""
        self.assertEqual('nt', os.name)
        self.assertEqual('win32', sys.platform)

    def test_imported_from_the_zip(self):
        """sys.path[0] is the .pex itself, and this module came out of it."""
        self.assertTrue(sys.argv[0].endswith('.pex.exe'), sys.argv[0])
        self.assertIn('.pex.exe', __file__)

    def test_reads_a_data_file(self):
        """Data files land beside the .pex rather than inside it."""
        with open('test/windows/python/data.txt') as f:
            self.assertEqual('hello from a data file', f.read().strip())

    def test_third_party_import(self):
        """Third-party code is imported from a directory inside the zip, by a meta path hook.

        Setting that hook up scans the zip for distribution metadata, whose member names are
        always /-separated whoever wrote them. Building the pattern for that out of os.sep made
        it a backslash here, which the regex compiler read as an escape - so every .pex died on
        startup, before any of its own code ran.
        """
        import six
        self.assertTrue(six.PY3)


if __name__ == '__main__':
    unittest.main()
