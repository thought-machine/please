subinclude('//build_defs:multiversion_wheel')
package(
    default_python_interpreter = 'python3',
    default_visibility = ['PUBLIC'],
    python_wheel_repo = 'https://get.please.build/third_party/python/py3',
)

REPOS = [
    'https://get.please.build/third_party/python/py27',
    'https://get.please.build/third_party/python/py34',
    'https://get.please.build/third_party/python/py35',
    'https://get.please.build/third_party/python/py36',
]

python_wheel(
    name = 'pex',
    version = '1.2.8',
    hashes = ['sha1: 15be967c06ab9686fb77397c0480345e75d6f0cc'],
)

python_wheel(
    name = 'xmlrunner',
    package_name = 'unittest_xml_reporting',
    version = '1.11.0',
    hashes = ['sha1: 62e71134eb068413e40fe4d973451c0e4931db5a'],
    deps = [':six'],
)

python_wheel(
    name = 'six',
    version = '1.10.0',
    outs = ['six.py'],
    hashes = ['sha1: 460beba667fbd2b7a050df6a456e0269260b385a'],
)

python_wheel(
    name = 'pkg_resources',
    package_name = 'setuptools',
    version = '33.1.1',
    hashes = ['sha1: 05f20a29a5dff7dd5b908afcf37c2b1bea8d3db6'],
)

python_wheel(
    name = 'requests',
    version = '2.5.0',
    hashes = [
        'sha1: e43b552c4a6063d32b8c9bf742fb3e9ce579c4d8',
    ],
    test_only = True,  # Not used by plz itself.
)

python_wheel(
    name = 'dateutil',
    package_name = 'python_dateutil',
    version = '2.4.0',
    hashes = [
        'sha1: 62b97c332a2c118b8957d324d7ad3cbcc1f4f9eb',
    ],
    test_only = True,  # Not used by plz itself.
    deps = [':six'],
)

python_wheel(
    name = 'protobuf',
    version = '3.4.0',
    outs = ['google'],
    deps = [':six'],
)

python_wheel(
    name = 'pycparser',
    version = '2.18',
)

python_multiversion_wheel(
    name = 'coverage',
    version = '4.3.4',
    repos = REPOS,
)

python_multiversion_wheel(
    name = 'cffi',
    version = '1.10.0',
    repos = REPOS,
    subdir = 'cffi',
)
