go_library(
    name="ide",
    srcs=glob(["*.go"], exclude=["*_test.go"]),
    deps=[
        "//src/core",
    ],
    visibility = ["PUBLIC"],
)

go_test(
    name="intellij_test",
    srcs=[
        "intellij_test.go",
    ],
    deps=[
        ":ide",
        "//third_party/go:testify",
    ],
)