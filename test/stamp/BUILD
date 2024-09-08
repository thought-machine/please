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

if is_platform(os = "linux", arch = "amd64"):
    go_binary(
        name = "stamp_static",
        srcs = ["main.go"],
        definitions = {
            "github.com/thought-machine/please/test/stamp/lib.GitRevision": "$SCM_REVISION",
            "github.com/thought-machine/please/test/stamp/lib.GitDescribe": "$SCM_DESCRIBE",
        },
        stamp = True,
        static = True,
        deps = ["//test/stamp/lib"],
    )

    sh_test(
        name = "stamp_test_static",
        src = "stamp_test.sh",
        data = [":stamp_static"],
    )
