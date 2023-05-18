subinclude("//build_defs:benchmark")

go_library(
    name = "core",
    srcs = [
        "atomic_bool.go",
        "atomic_float32.go",
        "build_env.go",
        "build_input.go",
        "build_label.go",
        "build_target.go",
        "cache.go",
        "command_replacements.go",
        "config.go",
        "config_flags.go",
        "cycle_detector.go",
        "graph.go",
        "lock.go",
        "package.go",
        "previous_op.go",
        "resources.go",
        "stamp.go",
        "state.go",
        "subrepo.go",
        "test_results.go",
        "utils.go",
        "version.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_cespare_xxhash_v2//:v2",
        "///third_party/go/github.com_coreos_go-semver//semver",
        "///third_party/go/github.com_google_shlex//:shlex",
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "///third_party/go/github.com_pkg_xattr//:xattr",
        "///third_party/go/github.com_please-build_gcfg//:gcfg",
        "///third_party/go/github.com_please-build_gcfg//types",
        "///third_party/go/github.com_shirou_gopsutil_v3//cpu",
        "///third_party/go/github.com_shirou_gopsutil_v3//mem",
        "///third_party/go/github.com_thought-machine_go-flags//:go-flags",
        "///third_party/go/github.com_zeebo_blake3//:blake3",
        "///third_party/go/golang.org_x_sync//errgroup",
        "//src/cli",
        "//src/cli/logging",
        "//src/cmap",
        "//src/fs",
        "//src/process",
        "//src/scm",
    ],
)

go_test(
    name = "core_test",
    srcs = [
        "build_env_test.go",
        "build_input_test.go",
        "build_label_fuzz_test.go",
        "build_label_test.go",
        "build_target_benchmark_test.go",
        "build_target_test.go",
        "command_replacements_test.go",
        "config_test.go",
        "cycle_detector_test.go",
        "graph_benchmark_test.go",
        "graph_test.go",
        "label_parse_test.go",
        "lock_test.go",
        "package_test.go",
        "previous_op_test.go",
        "stamp_test.go",
        "state_test.go",
        "subrepo_test.go",
        "test_results_test.go",
        "utils_benchmark_test.go",
        "utils_test.go",
    ],
    data = ["test_data"],
    filter_srcs = False,  # As above
    deps = [
        ":core",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "///third_party/go/github.com_thought-machine_go-flags//:go-flags",
        "//src/cli",
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
