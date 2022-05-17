subinclude("//build_defs:benchmark")

go_library(
    name = "core",
    srcs = glob(
        ["*.go"],
        exclude = [
            "*_benchmark.go",
            "*_test.go",
        ],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/cli/logging",
        "//src/cmap",
        "//src/fs",
        "//src/process",
        "//src/scm",
        "//third_party/go:blake3",
        "//third_party/go:deferred_regex",
        "//third_party/go:errgroup",
        "//third_party/go:gcfg",
        "//third_party/go:go-flags",
        "//third_party/go:godirwalk",
        "//third_party/go:psutil",
        "//third_party/go:semver",
        "//third_party/go:shlex",
        "//third_party/go:xattr",
        "//third_party/go:xxhash",
    ],
)

go_test(
    name = "config_test",
    srcs = ["config_test.go"],
    data = glob(["test_data/*.plzconfig*"]),
    filter_srcs = False,  # As above
    deps = [
        ":core",
        "//src/cli",
        "//third_party/go:go-flags",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "label_parse_test",
    srcs = ["label_parse_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "graph_test",
    srcs = ["graph_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "state_test",
    srcs = ["state_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "build_target_test",
    srcs = ["build_target_test.go"],
    data = ["test_data"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "package_test",
    srcs = ["package_test.go"],
    data = ["test_data"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "build_env_test",
    srcs = ["build_env_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "utils_test",
    srcs = ["utils_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "lock_test",
    srcs = ["lock_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "previous_op_test",
    srcs = ["previous_op_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "build_label_test",
    srcs = [
        "build_label_fuzz_test.go",
        "build_label_test.go",
    ],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "test_results_test",
    srcs = ["test_results_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "build_input_test",
    srcs = ["build_input_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "subrepo_test",
    srcs = ["subrepo_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "command_replacements_test",
    srcs = ["command_replacements_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "stamp_test",
    srcs = ["stamp_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "cycle_detector_test",
    srcs = ["cycle_detector_test.go"],
    deps = [
        ":core",
        "//third_party/go:testify",
    ],
)

benchmark(
    name = "build_target_benchmark",
    srcs = ["build_target_benchmark_test.go"],
    deps = [":core"],
)

benchmark(
    name = "utils_benchmark",
    srcs = ["utils_benchmark_test.go"],
    deps = [":core"],
)

benchmark(
    name = "graph_benchmark",
    srcs = ["graph_benchmark_test.go"],
    deps = [":core"],
)
