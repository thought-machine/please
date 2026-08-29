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
    hashes = ["4721f391ed90541fddacab5acf947aa0d3dc7d27b2e1e8eda2be8970586c3274"],
    version = "1.17.0",
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
    hashes = ["62f22742b58a1a33014a2b6b706588a8d7e2a88ae7bd1a6ebe8c992928483775"],
    version = "2026.7.22",
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
    hashes = ["9fb4c81ebbb1ce9531cce37674bbc6f1360472bc18ca9a553ede278ef7276897"],
    version = "2.7.0",
)

python_wheel(
    name = "colorlog",
    hashes = ["732c191ebbe9a353ec160d043d02c64ddef9028de8caae4cfa8bd49b6afed53e"],
    version = "4.1.0",
)

python_wheel(
    name = "absl",
    package_name = "absl_py",
    hashes = ["0f17b89f2a4eaaedc4f28c622998aa690564b3012a396a4ffad0821007fe03ba"],
    version = "2.5.0",
    deps = [":six"],
)

pip_library(
    name = "progress",
    version = "1.6.1",
    licences = ["ISC"],
    hashes = ["34ce5c7402e39aa9b88b988bc8559bd35b22c589d18b480251c7c72e15eba944"],
)
