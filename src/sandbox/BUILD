if CONFIG.OS == "linux":
    cgo_library(
        name = "sandbox",
        srcs = ["sandbox_linux.go"],
        hdrs = ["sandbox.h"],
        c_srcs = ["sandbox_linux.c"],
        visibility = ["//src/..."],
        deps = [
            "//src/core",
            "//src/process",
        ],
    )
else:
    go_library(
        name = "sandbox",
        srcs = ["sandbox_other.go"],
        visibility = ["//src/..."],
        deps = [
            "//src/process",
        ],
    )
