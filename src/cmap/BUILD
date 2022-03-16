go_library(
    name = "cmap",
    srcs = ["cmap.go", "hash.go"],
    visibility = ["PUBLIC"],
)

go_test(
    name = "cmap_test",
    srcs = ["cmap_test.go"],
    deps = [
        ":cmap",
        "//third_party/go:testify",
    ],
)

go_benchmark(
    name = "cmap_benchmark",
    srcs = ["cmap_test.go"],
    deps = [
        ":cmap",
        "//third_party/go:testify",
    ],
)

go_benchmark(
    name = "hash_benchmark",
    srcs = ["hash_test.go"],
    deps = [
        ":cmap",
        "//third_party/go:testify",
    ],
)
