go_library(
    name = "intellij",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//third_party/go:logging",
    ],
)

go_test(
    name = "intellij_test",
    srcs = [
        "intellij_test.go",
    ],
    deps = [
        ":intellij",
        "//third_party/go:testify",
    ],
)

go_test(
    name = "library_test",
    srcs = [
        "library_test.go",
    ],
    deps = [
        ":intellij",
        "//third_party/go:testify",
    ],
)
