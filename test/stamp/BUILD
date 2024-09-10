go_binary(
    name = "stamp",
    srcs = ["main.go"],
    definitions = {
        "github.com/thought-machine/please/test/stamp/lib.GitRevision": "$SCM_REVISION",
        "github.com/thought-machine/please/test/stamp/lib.GitDescribe": "$SCM_DESCRIBE",
    },
    stamp = True,
    deps = ["//test/stamp/lib"],
)

sh_test(
    name = "stamp_test",
    src = "stamp_test.sh",
    data = [":stamp"],
)
