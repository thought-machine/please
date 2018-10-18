go_library(
    name="intellij",
    srcs=glob(["*.go"], exclude=["*_test.go"]),
    deps=[
        "//src/core",
        "//third_party/go:logging",
    ],
    visibility = ["PUBLIC"],
)

go_test(
    name="intellij_test",
    srcs=[
        "intellij_test.go",
    ],
    deps=[
        ":intellij",
        "//third_party/go:testify",
    ],
)

go_test(
    name="library_test",
    srcs=[
        "library_test.go",
    ],
    deps=[
        ":intellij",
        "//third_party/go:testify",
    ],
)

go_test(
    name="module_test",
    srcs=[
        "module_test.go",
    ],
    deps=[
        ":intellij",
        "//third_party/go:testify",
    ],
)

go_test(
    name="project_test",
    srcs=[
        "project_test.go",
    ],
    deps=[
        ":intellij",
        "//third_party/go:testify",
    ],
)