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
    hashes = ["2cd93ac244981391a35b1e0017eecf90c8a962cd67cfde1c947303025961e055"],
    version = "1.14.0",
    use_package_root=True,
)

python_wheel(
    name = "certifi",
    hashes = ["e553ccb88a45bbef1d25ae65ad32b86e0ad3c6eff94f4b50a3976fccc8e49ab6"],
    version = "2019.11.28",
    use_package_root=True,
)

