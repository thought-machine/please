subinclude("//build_defs:multiversion_wheel")

package(
    default_visibility = ["PUBLIC"],
    python_wheel_repo = "https://get.please.build/third_party/python/py3",
)

python_wheel(
    name = "xmlrunner",
    package_name = "unittest_xml_reporting",
    hashes = ["62e71134eb068413e40fe4d973451c0e4931db5a"],
    version = "1.11.0",
    deps = [":six"],
)

python_wheel(
    name = "six",
    outs = ["six.py"],
    hashes = ["5a5d393fb2591037184a11b8dc40dc677bfd8c53"],
    version = "1.14.0",
)

python_wheel(
    name = "requests",
    hashes = [
        "b05a0f11b8238e26d6fe217e730dd7334fee64de",
    ],
    version = "2.23.0",
)

python_wheel(
    name = "colorlog",
    hashes = ["293a42e5e138b8f93892e2f193923b612fd9bdb7"],
    version = "4.1.0",
)

python_wheel(
    name = "dateutil",
    package_name = "python_dateutil",
    hashes = [
        "62b97c332a2c118b8957d324d7ad3cbcc1f4f9eb",
    ],
    test_only = True,  # Not used by plz itself.
    version = "2.4.0",
    deps = [":six"],
)

python_wheel(
    name = "protobuf",
    outs = ["google"],
    version = "3.6.1",
    deps = [":six"],
)

if CONFIG.ARCH == "amd64":
    python_multiversion_wheel(
        name = "coverage",
        repos = [
            "https://get.please.build/third_party/python/py27",
            "https://get.please.build/third_party/python/py34",
            "https://get.please.build/third_party/python/py35",
            "https://get.please.build/third_party/python/py36",
            "https://get.please.build/third_party/python/py37",
        ],
        version = "4.3.4",
    )
else:
    python_wheel(
        name = "coverage",
        hashes = ["sha1: 519e737035a8cc3349b66973e8a9c69f1ba1102a"],
        version = "4.3.4",
    )

python_wheel(
    name = "attrs",
    outs = ["attr"],
    hashes = ["ae202715f80ebff64bf8f03ba14fa949af4ca948"],
    version = "19.3.0",
)

python_wheel(
    name = "pluggy",
    hashes = ["aa20002cb7c9a67a06534eb958ad1d209b0bdb09"],
    version = "0.13.1",
    deps = [":importlib_metadata"],
)

python_wheel(
    name = "importlib_metadata",
    outs = [
        "importlib_metadata",
        "importlib_metadata-1.5.0.dist-info",
    ],
    version = "1.5.0",
    hashes = ["613ee6df7b39cb1e16d034e9d0b59b586446e890"],
    deps = [":zipp"],
)

python_wheel(
    name = "zipp",
    outs = ["zipp.py"],
    version = "3.1.0",
    hashes = ["9715ca014bb6031527514627c128a03c045d45ea"],
)

python_wheel(
    name = "py",
    hashes = ["f1b2947b1c7dc2310c931034ad8d39de4e390fa2"],
    version = "1.8.1",
)

python_wheel(
    name = "funcsigs",
    hashes = ["de18f4fe57b15873bae7b98ee464a4b7bd6b273a"],
    version = "1.0.2",
)

python_wheel(
    name = "pkg_resources",
    package_name = "setuptools",
    hashes = ["05f20a29a5dff7dd5b908afcf37c2b1bea8d3db6"],
    version = "33.1.1",
)

python_wheel(
    name = "pytest",
    outs = [
        "pytest",
        "_pytest",
    ],
    hashes = ["975a427a35084c82ff6661fcb7594201cb74a5a8"],
    version = "5.3.5",
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
        "behave",
    ],
    hashes = ["f0908034da87e831cc10cc972db7f0de6de77ba7"],
    version = "1.2.6",
    deps = [
        ":colorama",
        ":enum34",
        ":parse",
        ":parse_type",
        ":six",
        ":traceback2",
        ":win_unicode_console",
    ],
)

python_wheel(
    name = "parse",
    outs = [
        "parse.py",
    ],
    hashes = ["1ab3c6f543b8d9170c88b176676580612d76ac3d"],
    version = "1.15.0",
)

python_wheel(
    name = "parse_type",
    hashes = ["8ab5faacc86a2369f791d354ba67d3c097263533"],
    version = "0.5.2",
)

python_wheel(
    name = "traceback2",
    hashes = ["2c4b215a5e4c352d785a8977c3afb1a73f79153b"],
    licences = ["PSF"],
    version = "1.4.0",
)

python_wheel(
    name = "enum34",
    outs = ["enum"],
    hashes = ["404088628f24d02c91a5aff3ba3ad3a659337d83"],
    version = "1.1.9",
)

python_wheel(
    name = "win_unicode_console",
    hashes = ["de9e0946c0f0774cbb6c9e340096da07a2b8b4db"],
    version = "0.5",
)

python_wheel(
    name = "colorama",
    hashes = ["eddd3cddde1091a66ca2091309d579579d33dcde"],
    version = "0.4.3",
)

python_wheel(
    name = "absl",
    package_name = "absl_py",
    hashes = ["b86e848ba75a307195bd8f711c1ba4d696a30816"],
    version = "0.9.0",
    deps = [":six"],
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

pip_library(
    name = "scipy",
    test_only = True,
    version = "1.1.0",
    zip_safe = False,
    deps = [":numpy"],
)
