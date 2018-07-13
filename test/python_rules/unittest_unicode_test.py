import unittest


class UnittestUnicodeTest(unittest.TestCase):
    def test_unicode_latin(self):
        self.assertEqual('kérem', 'kérem')

    def test_unicode_emoji(self):
        self.assertEqual('✂', '✂')
        self.assertNotEqual('✔', '❄')
