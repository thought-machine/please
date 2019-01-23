cgo_library(
    name = "cgo",
    srcs = ["cgo.go"],
    go_srcs = ["go.go"],
    c_srcs = ["cgo.c"],
    hdrs = ["cgo.h"],
)

cgo_test(
    name = "cgo_test",
    srcs = ["cgo_test.go"],
    deps = [
        ":cgo",
        "//third_party/go:testify",
    ],
)
