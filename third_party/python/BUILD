package(default_visibility = ['PUBLIC'])

pip_library(
    name = 'pex',
    version = '1.1.1',
    hashes = ['sha1: 74d1cc64aab1d8ae5bea69822d690ef0eb4ecd53'],
    patch = [
        'never_cache_pex.patch',
        'bytecode_timestamps.patch',
    ],
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
    version = '1.9.0',
    outs = ['six.py'],
    hashes = ['sha1: 0c31ab7cf1a2761efa32d9a7e891ddeadc0d8673'],
)

pip_library(
    name = 'pkg_resources',
    package_name = 'setuptools',
    version = '6.1',
    outs = ['pkg_resources.py'],
    hashes = ['sha1: 33c8ea9686a15ca09ce96656d4a2074d75235dfe'],
)

pip_library(
    name = 'coverage',
    version = '4.0.3',
    patch = 'coverage_pex.patch',
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
    version = '3.0.0b1.post2',
    outs = ['google'],
    hashes = [
        'sha1: 4b74638c35f417ab33c3fe3a337f0bb18f0ff1ab',
        'sha1: 6d4c66f719238c058d3d5d1fbe7b2484eb0fe89d',
        'sha1: 4b74638c35f417ab33c3fe3a337f0bb18f0ff1ab',
    ],
    deps = [':six'],
)

pip_library(
    name = 'enum',
    package_name = 'enum34',
    version = '1.0.4',
    outs = ['enum'],
    hashes = [
        'sha1: ce61fd8ab8ec1390ce3c42d3d042d7dded9ed205',
    ],
)

pip_library(
    name = 'futures',
    version = '2.2.0',
    outs = [
        'concurrent',
        'futures',
    ],
    hashes = [
        'sha1: 3bb16f7c2ad9e13623349bd3f876c02d7e2e39f7',
    ],
)
