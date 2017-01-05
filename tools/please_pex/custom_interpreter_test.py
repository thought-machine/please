"""Test that our pexer is capable of building .pex files with custom interpreters."""

import platform
import unittest


class CustomInterpreterTest(unittest.TestCase):

    def testInterpreterIsPyPy(self):
        """Test that this is being run with PyPy."""
        self.assertEqual('PyPy', platform.python_implementation())


if __name__ == '__main__':
    unittest.main()
