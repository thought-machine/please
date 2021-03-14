go_library(
    name = "glob",
    srcs = ["glob.go"],
    visibility = ["//src/..."],
)

go_test(
    name = "glob_test",
    srcs = ["glob_test.go"],
    deps = [
        ":glob",
        "//third_party/go:testify",
    ],
)
