GO_CI_LINT_VERSION = "1.54.2"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % GO_CI_LINT_VERSION],
    extract = True,
    hashes = [
        "925c4097eae9e035b0b052a66d0a149f861e2ab611a4e677c7ffd2d4e05b9b89",  # darwin-amd64
        "7b33fb1be2f26b7e3d1f3c10ce9b2b5ce6d13bb1d8468a4b2ba794f05b4445e1",  # darwin-arm64
        "01301af199161c5ea33a0350581e1bed43289a56b2fbed7539f99fca2f898278",  # freebsd-amd64
        "17c9ca05253efe833d47f38caf670aad2202b5e6515879a99873fabd4c7452b3",  # linux-amd64
        "a9f14b33473c65fcfbf411ec054b53a87dbb849f4e09ee438f1ee76dbf3f3d4e",  # linux-arm64
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        GO_CI_LINT_VERSION,
        GO_CI_LINT_VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)

version = "1.1.1"

remote_file(
    name = "puku",
    binary = True,
    url = f"https://github.com/please-build/puku/releases/download/v{version}/puku-{version}-{CONFIG.OS}_{CONFIG.ARCH}",
)
