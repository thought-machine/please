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
    name = 'xmlrunner',
    package_name = 'unittest_xml_reporting',
    version = '1.11.0',
    hashes = ['sha1: 62e71134eb068413e40fe4d973451c0e4931db5a'],
    deps = [':six'],
)

python_wheel(
    name = 'six',
    version = '1.11.0',
    outs = ['six.py'],
    hashes = ['sha1: a96bbdb19bd254689e8687cef85593bc6b3d9703'],
)

python_wheel(
    name = 'requests',
    version = '2.5.0',
    hashes = [
        'sha1: e43b552c4a6063d32b8c9bf742fb3e9ce579c4d8',
    ],
    test_only = True,
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

python_wheel(
    name = 'attrs',
    version = '17.3.0',
    outs = ['attr'],
    hashes = ['sha1: 8c10e615550922e4cd9a52fe7166339c8d4615fd'],
)

python_wheel(
    name = 'pluggy',
    version = '0.6.0',
    hashes = ['sha1: 8413e603d74392f027e8a2e87a589a4f055f2e18'],
)

python_wheel(
    name = 'py',
    version = '1.5.2',
    hashes = ['sha1: f63c59196e88ce403e528630a76fb3f755157625'],
)

python_wheel(
    name = 'funcsigs',
    version = '1.0.2',
    hashes = ['sha1: de18f4fe57b15873bae7b98ee464a4b7bd6b273a'],
)

python_wheel(
    name = 'pytest',
    version = '3.3.0',
    outs = [
        '_pytest',
        'pytest.py',
    ],
    hashes = ['sha1: 27dbe6214cab8205259396111ed9465267de26da'],
    deps = [
        ':attrs',
        ':funcsigs',
        ':pluggy',
        ':py',
        ':six',
    ],
)
