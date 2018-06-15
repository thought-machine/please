subinclude("//build_defs:multiversion_wheel")

package(
    default_visibility = ["PUBLIC"],
    python_wheel_repo = "https://get.please.build/third_party/python/py3",
)

REPOS = [
    "https://get.please.build/third_party/python/py27",
    "https://get.please.build/third_party/python/py34",
    "https://get.please.build/third_party/python/py35",
    "https://get.please.build/third_party/python/py36",
]

python_wheel(
    name = "xmlrunner",
    package_name = "unittest_xml_reporting",
    hashes = ["sha1: 62e71134eb068413e40fe4d973451c0e4931db5a"],
    version = "1.11.0",
    deps = [":six"],
)

python_wheel(
    name = "six",
    outs = ["six.py"],
    hashes = ["sha1: a96bbdb19bd254689e8687cef85593bc6b3d9703"],
    version = "1.11.0",
)

python_wheel(
    name = "requests",
    hashes = [
        "sha1: e43b552c4a6063d32b8c9bf742fb3e9ce579c4d8",
    ],
    test_only = True,
    version = "2.5.0",
)

python_wheel(
    name = "dateutil",
    package_name = "python_dateutil",
    hashes = [
        "sha1: 62b97c332a2c118b8957d324d7ad3cbcc1f4f9eb",
    ],
    test_only = True,  # Not used by plz itself.
    version = "2.4.0",
    deps = [":six"],
)

python_wheel(
    name = "protobuf",
    outs = ["google"],
    version = "3.4.0",
    deps = [":six"],
)

python_wheel(
    name = "pycparser",
    version = "2.18",
)

python_multiversion_wheel(
    name = "coverage",
    repos = REPOS,
    version = "4.3.4",
)

python_wheel(
    name = "attrs",
    outs = ["attr"],
    hashes = ["sha1: 8c10e615550922e4cd9a52fe7166339c8d4615fd"],
    version = "17.3.0",
)

python_wheel(
    name = "pluggy",
    hashes = ["sha1: 8413e603d74392f027e8a2e87a589a4f055f2e18"],
    version = "0.6.0",
)

python_wheel(
    name = "py",
    hashes = ["sha1: f63c59196e88ce403e528630a76fb3f755157625"],
    version = "1.5.2",
)

python_wheel(
    name = "funcsigs",
    hashes = ["sha1: de18f4fe57b15873bae7b98ee464a4b7bd6b273a"],
    version = "1.0.2",
)

python_wheel(
    name = "pkg_resources",
    package_name = "setuptools",
    hashes = ["sha1: 05f20a29a5dff7dd5b908afcf37c2b1bea8d3db6"],
    version = "33.1.1",
)

python_wheel(
    name = "pytest",
    outs = [
        "_pytest",
        "pytest.py",
    ],
    hashes = ["sha1: 27dbe6214cab8205259396111ed9465267de26da"],
    version = "3.3.0",
    deps = [
        ":attrs",
        ":funcsigs",
        ":pkg_resources",
        ":pluggy",
        ":py",
        ":six",
    ],
)

python_wheel(
  name = "behave",
  outs = [
    "behave"
  ],
  version = "1.2.6",
  hashes = ["sha1: f0908034da87e831cc10cc972db7f0de6de77ba7"],
  deps = [
    ":six",
    ":parse",
    ":parse_type",
    ":traceback2",
    ":enum34",
    ":win_unicode_console",
    ":colorama"
  ],
)

python_wheel(
  name = "parse",
  version = "1.8.2",
  hashes = ['sha1: 87572632723b2adab2294adb0a58108576b75317'],
  outs = [
      "parse.py",
  ],
)

python_wheel(
    name = "parse_type",
    hashes = ['sha1: e928e6f948e6085d8141ecdafbb07df51828afda'],
    version = "0.4.2",
)

python_wheel(
    name = "traceback2",
    version = "1.4.0",
    hashes = ['sha1: 2c4b215a5e4c352d785a8977c3afb1a73f79153b'],
    licences = ['PSF']
)

python_wheel(
    name = "enum34",
    package_name = "enum",
    hashes = ['sha1: 95fa8672f6190129fa657c0b29ac8ff378cb9f8d'],
    outs = ['enum'],
    version = "1.1.6",
)

python_wheel(
    name = "win_unicode_console",
    hashes = ['sha1: de9e0946c0f0774cbb6c9e340096da07a2b8b4db'],
    version = "0.5",
)

python_wheel(
  name = "colorama",
  hashes = ['sha1: ee2bb9f67cd75e82dacab3ffcd92dd3e4d032a24'],
  version = "0.3.9",
)

pip_library(
    name = "numpy",
    test_only = True,
    version = "1.14.0",
    zip_safe = False,
)

pip_library(
    name = "tensorflow",
    test_only = True,
    version = "1.4.0",
    deps = [
        ":numpy",
        ":protobuf",
    ],
)

pip_library(
    name = "pytz",
    test_only = True,
    version = "2018.4",
)

pip_library(
    name = "pandas",
    test_only = True,
    version = "0.22.0",
    deps = [
        ":dateutil",
        ":numpy",
        ":pytz",
        ":six",
    ],
)
