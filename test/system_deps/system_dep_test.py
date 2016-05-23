"""Test on importing something as a system-level dependency."""

import unittest


class TestSystemDeps(unittest.TestCase):

    def test_can_import(self):
        """Test that we can import the system-level proto."""
        from test.system_deps import source_context_pb2
        sc = source_context_pb2.SourceContext()
        self.assertEqual('', sc.file_name)
