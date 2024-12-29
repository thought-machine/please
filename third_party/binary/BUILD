GO_CI_LINT_VERSION = "1.61.0"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)
