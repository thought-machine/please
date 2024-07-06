go_library(
    name = "cache",
    srcs = ["cache_client.go"],
    visibility = ["//src/remote:all"],
    deps = [
        "///third_party/go/github.com_bazelbuild_remote-apis-sdks//go/pkg/client",
        "///third_party/go/github.com_bazelbuild_remote-apis-sdks//go/pkg/digest",
        "//src/cli/logging",
        "//src/remote/fs",
    ],
)
