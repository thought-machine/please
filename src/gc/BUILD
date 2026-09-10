go_library(
    name = "gc",
    srcs = ["gc.go"],
    pgo_file = "//:pgo",
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
    data = [":gc_test_data"],
    # So //test/windows can run this same binary, cross-built, under Wine.
    visibility = ["//test/windows:all"],
    deps = [
        ":gc",
        "///third_party/go/github.com_stretchr_testify//assert",
        "//src/core",
        "//src/fs",
    ],
)

# Exposed so //test/windows can put it beside the test binary when running it under Wine.
filegroup(
    name = "gc_test_data",
    srcs = ["test_data"],
    test_only = True,
    visibility = ["//test/windows:all"],
)
