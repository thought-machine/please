go_binary(
    name = "stamp",
    srcs = ["main.go"],
    deps = ["//test/stamp/lib"],
    stamp = True,
    definitions = {
        "github.com/thought-machine/please/test/stamp.GitRevision": "$SCM_REVISION",
    },
)
