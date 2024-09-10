GO_CI_LINT_VERSION = "1.60.3"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    hashes = [
        "faf60366f99bb4010b634a030c45eaf57baae6c0b7e10be151139871e3fef40e", # golangci-lint-1.60.3-darwin-amd64.tar.gz
        "deb0fbd0b99992d97808614db1214f57d5bdc12b907581e2ef10d3a392aca11f", # golangci-lint-1.60.3-darwin-arm64.tar.gz
        "4037af8122871f401ed874852a471e54f147ff8ce80f5a304e020503bdb806ef", # golangci-lint-1.60.3-linux-amd64.tar.gz
        "74782943b2d2edae1208be3701e0cafe62817ba90b9b4cc5ca52bdef26df12f9", # golangci-lint-1.60.3-linux-arm64.tar.gz
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)
