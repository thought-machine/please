go_library(
    name = "audit",
    srcs = [
        "audit.go",
    ],
    visibility = ["PUBLIC"],
    pgo_file = "//:pgo",
    deps = [
        "///third_party/go/github.com_google_uuid//:uuid",
        "//src/cli/logging",
        "//src/fs",
    ],
)
