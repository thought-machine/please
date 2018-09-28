go_library(
    name="scm",
    srcs=glob(['*.go'],
               exclude = ["*_test.go"],
    ),
    deps=[
        "//src/core",
        "//third_party/go:logging",
    ],
    visibility = ["PUBLIC"],
)