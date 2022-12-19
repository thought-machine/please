subinclude("///go//build_defs:cgo")

cgo_library(
    name = "cgo",
    srcs = ["cgo.go"],
    hdrs = ["cgo.h"],
    c_srcs = ["cgo.c"],
    go_srcs = ["go.go"],
)

cgo_test(
    name = "cgo_test",
    srcs = ["cgo_test.go"],
    deps = [
        ":cgo",
        "//third_party/go:testify",
    ],
)
