go_library(
    name = "fs",
    srcs = [
        "attr.go",
        "copy.go",
        "executable.go",
        "fs.go",
        "glob.go",
        "hash.go",
        "home.go",
        "sort.go",
        "walk.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = [
        "///third_party/go/github.com_karrick_godirwalk//:godirwalk",
        "///third_party/go/github.com_peterebden_go-deferred-regex//:go-deferred-regex",
        "///third_party/go/github.com_pkg_xattr//:xattr",
        "//src/cli/logging",
        "//src/process",
    ],
)

go_test(
    name = "fs_test",
    srcs = [
        "copy_test.go",
        "fs_test.go",
        "glob_integration_test.go",
        "glob_test.go",
        "hash_benchmark_test.go",
        "hash_test.go",
        "home_test.go",
        "sort_test.go",
    ],
    data = ["test_data"],
    deps = [
        ":fs",
        "///third_party/go/github.com_cespare_xxhash_v2//:v2",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "///third_party/go/github.com_zeebo_blake3//:blake3",
    ],
)

go_test(
    name = "glob_integration_test",
    srcs = ["glob_integration_test.go"],
    # This must remain a glob because the tests rely on data being loaded in via globs
    data = glob(["test_data/**"]),
)

go_benchmark(
    name = "hash_benchmark",
    srcs = ["hash_benchmark_test.go"],
    deps = [
        ":fs",
        "///third_party/go/github.com_cespare_xxhash_v2//:v2",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
        "///third_party/go/github.com_zeebo_blake3//:blake3",
    ],
)
