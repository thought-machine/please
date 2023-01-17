subinclude("//build_defs:benchmark")

go_library(
    name = "asp",
    srcs = glob(
        ["*.go"],
        exclude = [
            "*_benchmark.go",
            "*_test.go",
            "*.bindata.go",
        ],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/cli/logging",
        "//src/cmap",
        "//src/core",
        "//src/fs",
        "//third_party/go:gcfg",
        "//third_party/go:promptui",
        "//third_party/go:semver2",
        "//third_party/go:x_exp",
    ],
)

go_test(
    name = "asp_test",
    srcs = glob(
        ["*_test.go"],
        exclude = [
            "util_test.go",
        ],
    ),
    data = ["test_data"],
    deps = [
        ":asp",
        "//rules",
        "//src/core",
        "//third_party/go:logging",
        "//third_party/go:testify",
    ],
)

benchmark(
    name = "interpreter_benchmark",
    srcs = ["interpreter_benchmark.go"],
    data = ["test_data"],
    deps = [
        ":asp",
        "//rules",
        "//src/core",
    ],
)
