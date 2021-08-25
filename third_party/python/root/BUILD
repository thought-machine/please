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

# this dependency lives here, so we have a dependency living outside
# the configured moduledir path

python_wheel(
    name = "six",
    outs = ["six.py"],
    hashes = ["5a5d393fb2591037184a11b8dc40dc677bfd8c53"],
    version = "1.14.0",
)

python_wheel(
    name = "certifi",
    hashes = ["adeac89bf3f6089e98e78b474cf4f5107b659e22"],
    version = "2019.11.28",
)

