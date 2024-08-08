GO_CI_LINT_VERSION = "1.58.2"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    hashes = [
        "9d8c372fabff0917a0502c86381e9de291bef274b5ccc8e6b849fccb257cfefd",  # darwin-amd64
        "99ba8037946fb3e3976a0004c93eb0df9f6efc50500f506665f1091ddf5aaba5",  # darwin-arm64
        "6236a423ea74cc0de61862da007a063f6187ad929bdb3fb54eb12cd689006a99",  # linux-amd64
        "549a714657e80942166ecb4327bfb21487af872945b50b155339b0b5cc3fa2f2",  # linux-arm64
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)
