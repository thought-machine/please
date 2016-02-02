import pkg_resources
import unittest


class SpecificOutTest(unittest.TestCase):

    @unittest.skip('Need third party module override')
    def test_python_module_is_importable(self):
        """Check that the Python module came through OK."""
        from test.proto_rules import test_pb2

    def test_no_go_files(self):
        """Test that there aren't any Go files in the .pex.

        If this fails the way the proto rules depend on specific files from the protoc rule
        probably isn't being selective enough.
        """
        self.assertFalse(pkg_resources.resource_exists('test.proto_rules', 'test.pb.go'))
