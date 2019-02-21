go_binary(
    name = "stamp",
    srcs = ["main.go"],
    deps = ["//test/stamp/lib"],
    stamp = True,
    definitions = {
        "github.com/thought-machine/please/test/stamp/lib.GitRevision": "$SCM_REVISION",
    },
)

sh_test(
    name = "stamp_test",
    src = "stamp_test.sh",
    data = [":stamp"],
)
