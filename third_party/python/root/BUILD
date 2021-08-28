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
    hashes = ["320b38e70e46b596b690a1306a50acab712de044"],
    version = "1.14.0",
    use_package_root=True,
)

python_wheel(
    name = "certifi",
    hashes = ["69dd32ce9d78c634665e2ee8cc5706aa1f6b2e62"],
    version = "2019.11.28",
    use_package_root=True,
)

