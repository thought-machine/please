cgo_library(
    name = "sandbox",
    srcs = ["sandbox.go"],
    hdrs = ["sandbox.h"],
    c_srcs = ["sandbox.c"],
    go_srcs = [
        "unshare.go",
    ],
    visibility = ["//src:please"],
    deps = [
        "//src/process",
    ],
)
