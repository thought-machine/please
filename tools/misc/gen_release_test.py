import unittest
from textwrap import dedent

from .gen_release import ReleaseGen


class GenReleaseTest(unittest.TestCase):

    def test_get_release_notes(self):
        """Tests that the changelog notes come out in a sensible way."""
        r = ReleaseGen('')
        r.version = '13.2.7'
        r.version_name = 'Version 13.2.7'
        expected = dedent("""
            This is Please v13.2.7

             * Disallowed adding empty outputs to a target (this never made sense and
               would only break things).
             * Langserver now supports autoformatting and completion for local files
               and argument names.
             * Fixed an issue with incorrect permissions when a rule marked as binary
               had a directory as an output (#478).

        """).lstrip()
        self.assertEqual(expected, '\n'.join(r.get_release_notes()))
