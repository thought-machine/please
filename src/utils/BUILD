go_library(
    name = "utils",
    srcs = glob(
        ["*.go"],
        exclude = [
            "wrapper_script.go",
            "stub.go",
            "*_test.go",
        ],
    ) + [":bindata"],
    visibility = ["PUBLIC"],
    deps = [
        "//src/cli",
        "//src/core",
        "//src/fs",
        "//third_party/go:levenshtein",
        "//third_party/go:logging",
        "//third_party/go:prompter",
    ],
)

go_bindata(
    name = "bindata",
    srcs = [
        "//tools/misc:completion_script",
        "//tools/misc:wrapper_script",
    ],
    prefix = "tools/misc",
)
