package(
    default_visibility = ["PUBLIC"],
    python = {
        "wheel_name_scheme": [
            "{url_base}/{package_name}-{version}-${{OS}}_${{ARCH}}.whl",
            "{url_base}/{package_name}-{version}.whl",
            "https://files.pythonhosted.org/packages/py3/{initial}/{package_name}/{package_name}-{version}-py3-none-any.whl",
            "https://files.pythonhosted.org/packages/py2.py3/{initial}/{package_name}/{package_name}-{version}-py2.py3-none-any.whl",
        ],
        "wheel_repo": "https://get.please.build/third_party/python/py3",
    },
)

python_wheel(
    name = "six",
    outs = ["six.py"],
    hashes = ["8abb2f1d86890a2dfb989f9a77cfcfd3e47c2a354b01111771326f8aa26e0254"],
    version = "1.16.0",
)

python_wheel(
    name = "requests",
    hashes = [
        "70761cfe03c773ceb22aa2f671b4757976145175cdfca038c02654d061d6dcc6",
    ],
    version = "2.32.3",
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
    name = "idna",
    hashes = ["a068a21ceac8a4d63dbfd964670474107f541babbd2250d61922f029858365fa"],
    version = "2.9",
)

python_wheel(
    name = "urllib3",
    hashes = ["a448b2f64d686155468037e1ace9f2d2199776e17f0a46610480d311f73e3472"],
    version = "2.2.2",
)

python_wheel(
    name = "colorlog",
    hashes = ["732c191ebbe9a353ec160d043d02c64ddef9028de8caae4cfa8bd49b6afed53e"],
    version = "4.1.0",
)

python_wheel(
    name = "absl",
    package_name = "absl_py",
    hashes = ["c106f6ef0ae86c1273b0858b40ee15b99fad1c223838387b9d11446a033bbcb1"],
    version = "0.9.0",
    deps = [":six"],
)

pip_library(
    name = "progress",
    version = "1.5",
)
