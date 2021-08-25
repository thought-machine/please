# Test for importing modules that were built with use_package_root=True
# When use_package_root is defined, they are unpacked in the root of
# the .pex file, hence imports work like regular imports.

import unittest


class PexImportUsingRootTest(unittest.TestCase):

    def test_import_six_using_root_packages(self):
        """Test importing six."""
        import six

    def test_import_certifi_using_root_package(self):
        """Test importing certifi."""
        import certifi

    def test_import_certifi_via_library(self):
        """Test importing certifi from another root library."""
        import certifi
        from pex_root_test_lib import get_certifi_module

        self.assertEqual(certifi, get_certifi_module())


if __name__ == '__main__':
    unittest.main()
