go_library(
    name = "command",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    deps = [
        "//src/core",
    ]
)

go_test(
    name = "command_test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":command",
        "//src/core",
        "//third_party/go:testify",
    ],
)