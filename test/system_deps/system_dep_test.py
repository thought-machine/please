"""Test on importing something as a system-level dependency."""

import unittest


class TestSystemDeps(unittest.TestCase):

    def test_can_import(self):
        """Test that we can import the system-level proto."""
        from test.system_deps import timestamp_pb2
        ts = timestamp_pb2.Timestamp()
        self.assertEqual(0, ts.seconds)
        self.assertEqual(0, ts.nanos)
