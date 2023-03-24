go_library(
    name = "gc",
    srcs = ["gc.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/cli/logging",
        "//src/core",
        "//src/parse/asp",
        "//src/scm",
    ],
)

go_test(
    name = "gc_test",
    srcs = [
        "gc_test.go",
        "rewrite_test.go",
    ],
    data = [
        "test_data",
    ],
    deps = [
        ":gc",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
        "//src/fs",
    ],
)
