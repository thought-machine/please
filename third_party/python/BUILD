package(default_visibility = ['PUBLIC'])

pip_library(
    name = 'pex',
    version = '1.2.8',
    hashes = ['sha1: 15be967c06ab9686fb77397c0480345e75d6f0cc'],
)

pip_library(
    name = 'xmlrunner',
    package_name = 'unittest-xml-reporting',
    version = '1.11.0',
    hashes = ['sha1: 62e71134eb068413e40fe4d973451c0e4931db5a'],
    deps = [':six'],
)

pip_library(
    name = 'six',
    version = '1.10.0',
    outs = ['six.py'],
    hashes = ['sha1: 460beba667fbd2b7a050df6a456e0269260b385a'],
)

pip_library(
    name = 'pkg_resources',
    package_name = 'setuptools',
    version = '33.1.1',
    hashes = ['sha1: 05f20a29a5dff7dd5b908afcf37c2b1bea8d3db6'],
)

pip_library(
    name = 'requests',
    version = '2.5.0',
    hashes = [
        'sha1: e43b552c4a6063d32b8c9bf742fb3e9ce579c4d8',
    ],
    test_only = True,  # Not used by plz itself.
)

pip_library(
    name = 'dateutil',
    package_name = 'python-dateutil',
    version = '2.4.0',
    hashes = [
        'sha1: 62b97c332a2c118b8957d324d7ad3cbcc1f4f9eb',
    ],
    test_only = True,  # Not used by plz itself.
    deps = [':six'],
)

pip_library(
    name = 'protobuf',
    version = '3.0.0',
    outs = ['google'],
    deps = [':six'],
)
