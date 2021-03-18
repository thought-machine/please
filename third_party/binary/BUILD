VERSION = "1.38.0"

remote_file(
    name = "golangci-lint",
    binary = True,
    exported_files = ["golangci-lint-%s-${OS}-${ARCH}/golangci-lint" % VERSION],
    extract = True,
    hashes = [
        "sha256: 9748c2697a65d2962106e74db6db6e3acc9066ec5ff0b81f47d803479e7b4994",  # darwin_arm64
        "sha256: a9b5eb572ce55ae900a3935640fa5e199729e784a6f058e8077a9a2126e00857",  # darwin_amd64
        "sha256: 97be8342ac9870bee003904bd8de25c0f3169c6b6238a013d6d6862efa5af992",  # linux_amd64
        "sha256: a05d18756522b2803ab53d49f5da074c298b69546691338809dbd16c7a4840fd",  # freebsd_amd64
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz" % (
        VERSION,
        VERSION,
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)
