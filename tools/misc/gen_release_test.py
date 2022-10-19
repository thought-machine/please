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

    def test_sha256(self):
        r = ReleaseGen('', dry_run=True)
        r.version = '13.2.7'

        path = "tools/misc/data/linux_amd64/release_asset_13.2.7"

        arch = r._arch(path)
        name = r.artifact_name(path)

        self.assertEqual("release_asset_13.2.7_linux_amd64", name)
        self.assertEqual("linux_amd64", arch)

        r.checksum(path)

        with open(f"{path}.sha256") as f:
            self.assertTrue(name in f.read())


