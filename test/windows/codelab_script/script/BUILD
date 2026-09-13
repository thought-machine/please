go_library(
    name = "script",
    srcs = [
        "classify.go",
        "parse.go",
        "plan.go",
        "sidecar.go",
    ],
    visibility = ["//test/windows/..."],
)

# The drift guard: runs against the real codelabs and the real sidecar, so a codelab edit that
# leaves a block undecided fails here, on Linux, in the default test pass.
go_test(
    name = "script_test",
    srcs = ["script_test.go"],
    data = [
        "test_data",
        "//docs/codelabs:codelab_md",
        "//test/windows:codelab_metadata",
    ],
    deps = [
        ":script",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
