subinclude("//build_defs:multiversion_wheel")

package(
    default_visibility = ["PUBLIC"],
    python = {
        "wheel_repo": "https://get.please.build/third_party/python/py3",
        "wheel_name_scheme": [
            "{url_base}/{package_name}-{version}-${{OS}}_${{ARCH}}.whl",
            "{url_base}/{package_name}-{version}.whl",
            "https://files.pythonhosted.org/packages/py3/{initial}/{package_name}/{package_name}-{version}-py3-none-any.whl",
            "https://files.pythonhosted.org/packages/py2.py3/{initial}/{package_name}/{package_name}-{version}-py2.py3-none-any.whl",
        ],
    },
)

python_wheel(
    name = "xmlrunner",
    package_name = "unittest_xml_reporting",
    hashes = ["530e703bf8f90a867ce42a524fc4cc99c50c0a1522ce7bd010a783698bd4cac9"],
    version = "1.11.0",
    deps = [":six"],
)

python_wheel(
    name = "six",
    outs = ["six.py"],
    hashes = ["8f3cd2e254d8f793e7f3d6d9df77b92252b52637291d0f0da013c76ea2724b6c"],
    version = "1.14.0",
)

python_wheel(
    name = "requests",
    hashes = [
        "43999036bfa82904b6af1d99e4882b560e5e2c68e5c4b0aa03b655f3d7d73fee",
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
    hashes = ["017c25db2a153ce562900032d5bc68e9f191e44e9a0f762f373977de9df1fbb3"],
    version = "2019.11.28",
)

python_wheel(
    name = "chardet",
    hashes = ["fc323ffcaeaed0e0a02bf4d117757b98aed530d9ed4531e3e15460124c106691"],
    version = "3.0.4",
)

python_wheel(
    name = "debugpy",
    hashes = ["f058c204341fd7ff800ee0edafc106ca0fb1c9857e8a8895a6e04cca3ddcb7bf"],
    licences = ["MIT"],
    version = "1.5.0",
)

python_wheel(
    name = "idna",
    hashes = ["a068a21ceac8a4d63dbfd964670474107f541babbd2250d61922f029858365fa"],
    version = "2.9",
)

python_wheel(
    name = "urllib3",
    hashes = ["2f3db8b19923a873b3e5256dc9c2dedfa883e33d87c690d9c7913e1f40673cdc"],
    version = "1.25.8",
)

python_wheel(
    name = "colorlog",
    hashes = ["732c191ebbe9a353ec160d043d02c64ddef9028de8caae4cfa8bd49b6afed53e"],
    version = "4.1.0",
)

python_wheel(
    name = "dateutil",
    package_name = "python_dateutil",
    hashes = [
        "5179fabb692e3ec5a6e994eb9a72a6f4498f19106774672fd7e664a7aa9f49c3",
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
    hashes = ["08a96c641c3a74e44eb59afb61a24f2cb9f4d7188748e76ba4bb5edfa3cb7d1c"],
    version = "19.3.0",
)

python_wheel(
    name = "pluggy",
    hashes = ["966c145cd83c96502c3c3868f50408687b38434af77734af1e9ca461a4081d2d"],
    version = "0.13.1",
    deps = [":importlib_metadata"],
)

python_wheel(
    name = "importlib_metadata",
    outs = [
        "importlib_metadata",
        "importlib_metadata-1.5.0.dist-info",
    ],
    hashes = ["b97607a1a18a5100839aec1dc26a1ea17ee0d93b20b0f008d80a5a050afb200b"],
    patch = "importlib_metadata.patch",
    version = "1.5.0",
    deps = [":zipp"],
)

python_wheel(
    name = "zipp",
    outs = ["zipp.py"],
    hashes = ["aa36550ff0c0b7ef7fa639055d797116ee891440eac1a56f378e2d3179e0320b"],
    version = "3.1.0",
)

python_wheel(
    name = "py",
    hashes = ["c20fdd83a5dbc0af9efd622bee9a5564e278f6380fffcacc43ba6f43db2813b0"],
    version = "1.8.1",
)

python_wheel(
    name = "funcsigs",
    hashes = ["330cc27ccbf7f1e992e69fef78261dc7c6569012cf397db8d3de0234e6c937ca"],
    version = "1.0.2",
)

python_wheel(
    name = "pkg_resources",
    package_name = "setuptools",
    hashes = ["c8b9f1a457949002e358fea7d3f2a1e1b94ddc0354b2e40afc066bf95d21bf7b"],
    version = "57.0.0",
)

python_wheel(
    name = "packaging",
    hashes = ["170748228214b70b672c581a3dd610ee51f733018650740e98c7df862a583f73"],
    version = "20.1",
)

python_wheel(
    name = "more_itertools",
    hashes = ["5dd8bcf33e5f9513ffa06d5ad33d78f31e1931ac9a18f33d37e77a180d393a7c"],
    version = "8.2.0",
)

python_wheel(
    name = "pytest",
    outs = [
        "_pytest",
        "pytest",
    ],
    hashes = ["5c0db86b698e8f170ba4582a492248919255fcd4c79b1ee64ace34301fb589a1"],
    version = "5.4.3",
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
    hashes = ["ebda1a6c9e5bfe95c5f9f0a2794e01c7098b3dde86c10a95d8621c5907ff6f1c"],
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
    hashes = ["2bb9c56c65bf609abbbef372bdcdd7b9c163745e7daf5e8217dba50a55850510"],
    version = "1.15.0",
)

python_wheel(
    name = "parse_type",
    hashes = ["089a471b06327103865dfec2dd844230c3c658a4a1b5b4c8b6c16c8f77577f9e"],
    version = "0.5.2",
)

python_wheel(
    name = "traceback2",
    hashes = ["8253cebec4b19094d67cc5ed5af99bf1dba1285292226e98a31929f87a5d6b23"],
    licences = ["PSF"],
    version = "1.4.0",
)

python_wheel(
    name = "enum34",
    outs = ["enum"],
    hashes = ["708aabfb3d5898f99674c390d360d59efdd08547019763622365f19e84a7fef4"],
    version = "1.1.9",
)

python_wheel(
    name = "win_unicode_console",
    hashes = ["7a7fb8bf98bcacb328d1f607065c0f89107c7e8f4b02caf12e5f02fb462512a6"],
    version = "0.5",
)

python_wheel(
    name = "colorama",
    hashes = ["7d73d2a99753107a36ac6b455ee49046802e59d9d076ef8e47b61499fa29afff"],
    version = "0.4.3",
)

python_wheel(
    name = "absl",
    package_name = "absl_py",
    hashes = ["c106f6ef0ae86c1273b0858b40ee15b99fad1c223838387b9d11446a033bbcb1"],
    version = "0.9.0",
    deps = [":six"],
)

python_wheel(
    name = "portalocker",
    hashes = ["874d6063c6ceb185fe4771da41b01872d2c56d292db746698f8ad7bf1833c905"],
    version = "1.7.0",
)

pip_library(
    name = "setuptools",
    test_only = True,
    version = "47.1.1",
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

pip_library(
    name = "cx_oracle",
    package_name = "cx-Oracle",
    version = "8.2.1",
)
