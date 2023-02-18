package(please_go_tool = "please_go")

go_library(
    name = "test",
    srcs = [
        "find_cover_vars.go",
        "gotest.go",
        "write_test_main.go",
    ],
    visibility = ["//tools/please_go/..."],
    deps = ["//tools/please_go/install/toolchain"],
)

go_test(
    name = "find_cover_vars_test",
    srcs = [
        "find_cover_vars_test.go",
        "write_test_main_test.go",
    ],
    data = [
        "test_data/binary/core.a",
        "test_data/binary/lock.go",
        "test_data/core.a",
        "test_data/lock.go",
        ":test_excluded_archive",
    ],
    deps = [
        ":test",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)

# Little tweak so we don't have to commit this file twice.
genrule(
    name = "test_excluded_archive",
    srcs = ["test_data/core.a"],
    outs = ["test_data/x/core.a"],
    cmd = 'cp $SRC "$OUT"',
)

go_test(
    name = "write_test_main_test",
    srcs = ["write_test_main_test.go"],
    data = glob(["test_data/**/*.go"]),
    deps = [
        ":test",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
