go_library(
    name = "test",
    srcs = glob(
        ["*.go"],
        exclude = [
            "*_stub.go",
            "*_test.go",
        ],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//src/metrics",
        "//src/utils",
        "//src/worker",
        "//third_party/go:docker",
        "//third_party/go:logging",
        "//third_party/go:tools",
    ],
)

go_test(
    name = "results_test",
    srcs = ["results_test.go"],
    data = glob(["test_data/*"]),
    deps = [
        ":test",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "test_step_test",
    srcs = ["test_step_test.go"],
    deps = [
        ":test",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "container_test",
    srcs = ["container_test.go"],
    container = True,
    data = ["test_data/container_data.txt"],
)

go_test(
    name = "coverage_test",
    srcs = ["coverage_test.go"],
    data = [
        "test_data/gcov_coverage.gcov",
        "test_data/go_coverage.txt",
        "test_data/go_coverage_2.txt",
        "test_data/go_coverage_3.txt",
        "test_data/istanbul_coverage.json",
        "test_data/istanbul_coverage_2.json",
        "test_data/python-coverage.xml",
    ],
    deps = [
        ":test",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "xml_results_test",
    srcs = ["xml_results_test.go"],
    deps = [
        ":test",
        "//third_party/go:testify",
    ],
)
