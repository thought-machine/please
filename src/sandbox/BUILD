go_library(
    name = "sandbox",
    srcs = [
        "sandbox_linux.go",
        "sandbox_other.go",
    ],
    pgo_file = "//:pgo",
    visibility = ["//src/..."],
    deps = [
        "///third_party/go/golang.org_x_sys//unix",
        "//src/core",
    ],
)
