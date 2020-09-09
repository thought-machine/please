"""Tests for namespaced packages that share common folders."""

import unittest


class NamespacedPackagesTest(unittest.TestCase):

    def test_namespaced_packages_are_importable(self):
        import google.common.api.auth
        import google.protobuf.message
        self.assertIsNotNone(auth.Provider)
        self.assertIsNotNone(message.Message)

