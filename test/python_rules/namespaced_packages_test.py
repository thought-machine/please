"""Tests for namespaced packages that share common folders."""

import unittest

class NamespacedPackagesTest(unittest.TestCase):

    def test_namespaced_packages_are_importable(self):
        from google.api import auth_pb2
        from google.protobuf import message
        self.assertIsNotNone(auth_pb2.AuthProvider)
        self.assertIsNotNone(message.Message)

