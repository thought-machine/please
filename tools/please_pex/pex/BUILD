go_library(
    name = "pex",
    srcs = [
        "pex.go",
        "stub.go",
    ],
    resources = glob([
        "*.py",
        "debuggers/*.py",
        "test_runners/*.py",
    ]),
    visibility = ["//tools/please_pex:all"],
    deps = ["//tools/please_pex/zip"],
)
