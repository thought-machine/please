# Test for importing certain modules with the new incremental pex
# rules. These have proven tricky, seemingly around requests doing
# "from . import utils" etc (which is perfectly fine, and was working
# previously, this just helps investigate & make sure it's fixed).

import unittest


class PexImportTest(unittest.TestCase):

    def test_import_requests(self):
        """Test importing Requests."""
        from third_party.python import requests

    def test_import_dateutil(self):
        """Test importing dateutil."""
        from third_party.python.dateutil import parser


if __name__ == '__main__':
    unittest.main()
