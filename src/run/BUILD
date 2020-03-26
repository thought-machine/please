go_library(
    name = "run",
    srcs = ["run_step.go"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/output",
        "//src/process",
        "//third_party/go:errgroup",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "run_test",
    srcs = ["run_test.go"],
    data = ["test_data"],
    deps = [
        ":run",
        "//third_party/go:testify",
    ],
)
