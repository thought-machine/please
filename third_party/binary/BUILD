# TODO(peterebden): Use the following once we fix #1098
# remote_file(
#     name = "golangci-lint",
#     out = "golangci-lint",
#     binary = True,
#     extract = True,
#     strip_prefix = "golangci-lint-1.37.1-${OS}-${ARCH}",
#     url = "https://github.com/golangci/golangci-lint/releases/download/v1.37.1/golangci-lint-1.37.1-%s-%s.tar.gz" % (
#         CONFIG.OS,
#         CONFIG.ARCH,
#     ),
# )

remote_file(
    name = "golangci-lint_download",
    url = "https://github.com/golangci/golangci-lint/releases/download/v1.37.1/golangci-lint-1.37.1-%s-%s.tar.gz" % (
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)

genrule(
    name = "golangci-lint",
    srcs = [":golangci-lint_download"],
    outs = ["golangci-lint"],
    binary = True,
    cmd = "$TOOL x $SRCS --strip_prefix golangci-lint-1.37.1-${OS}-${ARCH}",
    tools = [CONFIG.JARCAT_TOOL],
)
