import unittest
import os.path


class JsOutTest(unittest.TestCase):

    def test_has_js_file(self):
        """Test that there is a generated JS proto file.

        We don't have proper JS support yet, so this is just testing we can
        generate a JS proto file
        """
        self.assertTrue(os.path.isfile('test/proto_rules/js_test_pb.js'))
