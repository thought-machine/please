import unittest


class ImportlibMetadataTest(unittest.TestCase):

    def test_can_detect_version(self):
        # This fails  if .dist-info isn't  available for wheels downloaded from
        # pip
        from importlib_metadata import version
        self.assertEqual(version('importlib_metadata'), '0.23')
