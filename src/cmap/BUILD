go_library(
    name = "cmap",
    srcs = [
        "cmap.go",
        "hash.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["PUBLIC"],
    deps = ["///third_party/go/github.com_cespare_xxhash_v2//:v2"],
)

go_test(
    name = "cmap_test",
    srcs = [
        "cmap_test.go",
        "hash_test.go",
    ],
    deps = [
        ":cmap",
        "///third_party/go/github.com_cespare_xxhash_v2//:v2",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/golang.org_x_sync//errgroup",
    ],
)

go_benchmark(
    name = "cmap_benchmark",
    srcs = ["cmap_test.go"],
    deps = [
        ":cmap",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/golang.org_x_sync//errgroup",
    ],
)

go_benchmark(
    name = "hash_benchmark",
    srcs = ["hash_test.go"],
    deps = [
        ":cmap",
        "///third_party/go/github.com_cespare_xxhash_v2//:v2",
    ],
)
