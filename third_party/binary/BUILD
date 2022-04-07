GO_CI_LINT_VERSION = "1.45.2"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    hashes = [
        "sha256: c2b9669decc1b638cf2ee9060571af4e255f6dfcbb225c293e3a7ee4bb2c7217",  # darwin_arm64
        "sha256: 995e509e895ca6a64ffc7395ac884d5961bdec98423cb896b17f345a9b4a19cf",  # darwin_amd64
        "sha256: 595ad6c6dade4c064351bc309f411703e457f8ffbb7a1806b3d8ee713333427f",  # linux_amd64
        "sha256: 726cb045559b7518bafdd3459de70a0647c087eb1b4634627a4b2e95b1258580",  # freebsd_amd64
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)

GO_DEPS_VERSION = "v1.4.1"

# TODO(jpoole): this should support other platforms than just linux_amd64
remote_file(
    name = "go-deps",
    binary = True,
    url = f"https://github.com/Tatskaari/go-deps/releases/download/{GO_DEPS_VERSION}/go_deps_{GO_DEPS_VERSION}_{CONFIG.OS}_{CONFIG.ARCH}",
)
