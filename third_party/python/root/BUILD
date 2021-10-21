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
    hashes = ["8f3cd2e254d8f793e7f3d6d9df77b92252b52637291d0f0da013c76ea2724b6c"],
    version = "1.14.0",
    use_package_root=True,
)

python_wheel(
    name = "certifi",
    hashes = ["017c25db2a153ce562900032d5bc68e9f191e44e9a0f762f373977de9df1fbb3"],
    version = "2019.11.28",
    use_package_root=True,
)

