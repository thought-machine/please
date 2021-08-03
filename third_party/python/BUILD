subinclude("//build_defs:multiversion_wheel")

package(
    default_visibility = ["PUBLIC"],
    python_wheel_name_scheme = [
        "{url_base}/{package_name}-{version}-${{OS}}_${{ARCH}}.whl",
        "{url_base}/{package_name}-{version}.whl",
        "https://files.pythonhosted.org/packages/py3/{initial}/{package_name}/{package_name}-{version}-py3-none-any.whl",
        "https://files.pythonhosted.org/packages/py2.py3/{initial}/{package_name}/{package_name}-{version}-py2.py3-none-any.whl",
    ],
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
    deps = [
        ":certifi",
        ":chardet",
        ":idna",
        ":urllib3",
    ],
)

python_wheel(
    name = "certifi",
    hashes = ["adeac89bf3f6089e98e78b474cf4f5107b659e22"],
    version = "2019.11.28",
)

python_wheel(
    name = "chardet",
    hashes = ["f98c61fa8032f4122130ba8d99f3e3c9059f0237"],
    version = "3.0.4",
)

python_wheel(
    name = "idna",
    hashes = ["b5bf6680fefd37d19c3a50bfe104825c5d731ee5"],
    version = "2.9",
)

python_wheel(
    name = "urllib3",
    hashes = ["d804a9d60dada189e92ae5c48e35da7c06182a5b"],
    version = "1.25.8",
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
    version = "3.12.2",
    deps = [":six"],
)

if is_platform(
    arch = "amd64",
    os = "linux",
):
    urls = [
        "https://files.pythonhosted.org/packages/2f/19/4ebe9fe7006d46dd56eacd8cdc800b465590037bffeea17852520613cfaf/coverage-5.5-cp35-cp35m-manylinux2010_x86_64.whl",
        "https://files.pythonhosted.org/packages/42/37/a82863f91b41711203277ea286bc37915063f4d1be179ac34b591bf6d8a5/coverage-5.5-cp36-cp36m-manylinux2010_x86_64.whl",
        "https://files.pythonhosted.org/packages/16/e0/fc9f7bd9b84e6b41d0aad1a113e36714aac0c0a9b307aca5f9af443bc50f/coverage-5.5-cp37-cp37m-manylinux2010_x86_64.whl",
        "https://files.pythonhosted.org/packages/a4/3a/8f7b217265503eae2b0ea97e714e2709e1e84ee13cd3ca6abdff1e99e76c/coverage-5.5-cp38-cp38-manylinux2010_x86_64.whl",
        "https://files.pythonhosted.org/packages/a4/79/625f1ed5da2a69f52fb44e0b7ca1b470437ff502348c716005a98a71cd49/coverage-5.5-cp39-cp39-manylinux2010_x86_64.whl",
    ]
elif is_platform(
    arch = "amd64",
    os = "darwin",
):
    urls = [
        "https://files.pythonhosted.org/packages/9f/16/7e0972f8495f6a1b81cfa6579eead931d63dd445e8ecb3114b04a0e36af2/coverage-5.5-cp35-cp35m-macosx_10_9_x86_64.whl",
        "https://files.pythonhosted.org/packages/fd/2b/ab03276eb127f8ec7f1cf1499c77944321b125d89859ab51ee7d9f46475f/coverage-5.5-cp36-cp36m-macosx_10_9_x86_64.whl",
        "https://files.pythonhosted.org/packages/52/44/5df49f3b462a0f5818a2f6f206d6523ff21ff9b21c1eb2906f8a31aa321c/coverage-5.5-cp37-cp37m-macosx_10_9_x86_64.whl",
        "https://files.pythonhosted.org/packages/b6/26/b53bf0fef1b4bce6f7d61fef10fbf924d943987d4c9e53c394ecebff3673/coverage-5.5-cp38-cp38-macosx_10_9_x86_64.whl",
        "https://files.pythonhosted.org/packages/0d/8a/3b13c4e1f241a7083a4ee9986b969f0238f41dcd7a8990c786bc3b4b5b19/coverage-5.5-cp39-cp39-macosx_10_9_x86_64.whl",
    ]
else:
    urls = [
        "https://files.pythonhosted.org/packages/93/ac/02bc6a60304a8a58383386f7675f6ebae0d2f49f162dda318e57bd95c746/coverage-5.5-pp37-none-any.whl",
    ]

python_multiversion_wheel(
    name = "coverage",
    urls = urls,
    version = "5.5",
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
    hashes = ["f83974a5581882d6fd23df67e2c2c9628ca75109"],
    patch = "importlib_metadata.patch",
    version = "1.5.0",
    deps = [":zipp"],
)

python_wheel(
    name = "zipp",
    outs = ["zipp.py"],
    hashes = ["9715ca014bb6031527514627c128a03c045d45ea"],
    version = "3.1.0",
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
    hashes = ["184aa380e787226bd0940098ba12a2f6aeb5d000b6ac46fd35eb8f61cbf9aa1d"],
    version = "57.0.0",
)

python_wheel(
    name = "packaging",
    hashes = ["5dac7a1f3703c2eda994feb3148a8f13595200bb"],
    version = "20.1",
)

python_wheel(
    name = "more_itertools",
    hashes = ["a9731580196536ecc037ac342773382c46816451"],
    version = "8.2.0",
)

python_wheel(
    name = "pytest",
    outs = [
        "_pytest",
        "pytest",
    ],
    hashes = ["975a427a35084c82ff6661fcb7594201cb74a5a8"],
    version = "5.3.5",
    deps = [
        ":attrs",
        ":funcsigs",
        ":more_itertools",
        ":packaging",
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

python_wheel(
    name = "portalocker",
    hashes = ["9fa00efa3102a84ed5281de63189de813f0f83fd"],
    version = "1.7.0",
)

pip_library(
    name = "numpy",
    test_only = True,
    version = "1.18.4",
    zip_safe = False,
)

pip_library(
    name = "tensorflow",
    licences = ["Apache 2.0"],
    test_only = True,
    # the next version of TF breaks due to an upstream bug fixed in this pr:
    # https://github.com/tensorflow/tensorflow/pull/38663
    version = "1.13.1",
    deps = [
        ":absl",
        ":astor",
        ":gast",
        ":google-pasta",
        ":grpcio",
        ":keras_applications",
        ":keras_preprocessing",
        ":numpy",
        ":opt_einsum",
        ":protobuf",
        ":termcolor",
        ":wrapt",
    ],
)

pip_library(
    name = "keras_applications",
    test_only = True,
    version = "1.0.8",
    deps = [
        ":h5py",
    ],
)

pip_library(
    name = "opt_einsum",
    test_only = True,
    version = "3.2.1",
    deps = [
        ":numpy",
    ],
)

pip_library(
    name = "keras_preprocessing",
    test_only = True,
    version = "1.1.0",
    deps = [
        ":numpy",
        ":six",
    ],
)

pip_library(
    name = "grpcio",
    test_only = True,
    version = "1.32.0",
    deps = [
        ":six",
    ],
)

pip_library(
    name = "google-pasta",
    test_only = True,
    version = "0.2.0",
    deps = [
        ":six",
    ],
)

pip_library(
    name = "gast",
    test_only = True,
    version = "0.2.2",
)

pip_library(
    name = "astor",
    test_only = True,
    version = "0.8.1",
)

pip_library(
    name = "termcolor",
    test_only = True,
    version = "1.1.0",
)

pip_library(
    name = "wrapt",
    test_only = True,
    version = "1.12.1",
)

pip_library(
    name = "h5py",
    test_only = True,
    version = "2.10.0",
    deps = [
        ":numpy",
        ":six",
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
    version = "1.0.3",
    deps = [
        ":dateutil",
        ":numpy",
        ":pytz",
        ":six",
    ],
)

pip_library(
    name = "setuptools",
    test_only = True,
    version = "47.1.1",
)

pip_library(
    name = "scipy",
    test_only = True,
    version = "1.5.2",
    zip_safe = False,
    deps = [
        ":numpy",
        ":setuptools",
    ],
)

pip_library(
    name = "googleapis_common_protos",
    package_name = "googleapis-common-protos",
    test_only = True,
    version = "1.52.0",
    deps = [
        ":protobuf_pip",
    ],
)

pip_library(
    name = "protobuf_pip",
    package_name = "protobuf",
    test_only = True,
    version = "3.12.2",
    deps = [
        ":setuptools",
        ":six",
    ],
)

pip_library(
    name = "progress",
    version = "1.5",
)
