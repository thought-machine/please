go_library(
    name = "fs",
    srcs = [
        "fs.go",
        "glob.go",
        "hash.go",
        "home.go",
    ],
    visibility = ["PUBLIC"],
    deps = ["//third_party/go:godirwalk"],
)

go_test(
    name = "fs_test",
    srcs = ["fs_test.go"],
    data = ["test_data"],
    deps = [
        ":fs",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "hash_test",
    srcs = ["hash_test.go"],
    data = ["test_data"],
    deps = [
        ":fs",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "home_test",
    srcs = ["home_test.go"],
    deps = [
        ":fs",
        "//third_party/go:testify",
    ],
)
