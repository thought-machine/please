import base64
import unittest

MSG = 'UmljY2FyZG8gbGlrZXMgcGluZWFwcGxlIHBpenphCg=='


class TestExtraTestOutput(unittest.TestCase):

    def test_extra_output(self):
        with open('truth.txt', 'wb') as f:
            f.write(base64.b64decode(MSG))
