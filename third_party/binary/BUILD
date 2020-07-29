# TODO(peterebden): Use the following once we fix #1098
# remote_file(
#     name = "golangci-lint",
#     out = "golangci-lint",
#     binary = True,
#     extract = True,
#     hashes = [
#         "98b1eb7c74766079e1deebc3388c13db9bfa9fa0769046d786cf8d1553d7d68b",
#         "56b36d1b0a976caf2f49c06e4429aa6021ef0aba13a2e823e063b0605e1dd492",
#         "6a7c31abca3f51714e5ea1f0aae5dc78d72e7d57d07f02b1a1778219d5648e21",
#     ],
#     strip_prefix = "golangci-lint-1.29.0-${OS}-${ARCH}",
#     url = "https://github.com/golangci/golangci-lint/releases/download/v1.29.0/golangci-lint-1.29.0-%s-%s.tar.gz" % (
#         CONFIG.OS,
#         CONFIG.ARCH,
#     ),
# )

remote_file(
    name = "golangci-lint_download",
    hashes = [
        "98b1eb7c74766079e1deebc3388c13db9bfa9fa0769046d786cf8d1553d7d68b",
        "56b36d1b0a976caf2f49c06e4429aa6021ef0aba13a2e823e063b0605e1dd492",
        "6a7c31abca3f51714e5ea1f0aae5dc78d72e7d57d07f02b1a1778219d5648e21",
    ],
    url = "https://github.com/golangci/golangci-lint/releases/download/v1.29.0/golangci-lint-1.29.0-%s-%s.tar.gz" % (
        CONFIG.OS,
        CONFIG.ARCH,
    ),
)

genrule(
    name = "golangci-lint",
    srcs = [":golangci-lint_download"],
    outs = ["golangci-lint"],
    binary = True,
    cmd = "$TOOL x $SRCS --strip_prefix golangci-lint-1.29.0-${OS}-${ARCH}",
    tools = [CONFIG.JARCAT_TOOL],
)
