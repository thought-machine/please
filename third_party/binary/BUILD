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

PUKU_VERSION = "1.14.0"

remote_file(
    name = "puku",
    binary = True,
    url = f"https://github.com/please-build/puku/releases/download/v{PUKU_VERSION}/puku-{PUKU_VERSION}-{CONFIG.OS}_{CONFIG.ARCH}",
)
