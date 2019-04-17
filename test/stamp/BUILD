go_binary(
    name = "stamp",
    srcs = ["main.go"],
    definitions = {
        "github.com/thought-machine/please/test/stamp/lib.GitRevision": "$SCM_REVISION",
    },
    stamp = True,
    static = True,
    deps = ["//test/stamp/lib"],
)

sh_test(
    name = "stamp_test",
    src = "stamp_test.sh",
    data = [":stamp"],
)
