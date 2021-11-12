go_library(
    name = "lint",
    srcs = glob(["*.go"], exclude=["*_test.go"]),
    deps = [
        "//src/core",
        "//third_party/go:logging",
    ],
    visibility = ["//src/..."],
)
