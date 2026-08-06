go_test(
    name = "sandbox_test",
    srcs = ["sandbox_test.go"],
    data = {
        "please_sandbox": "//tools/sandbox:please_sandbox",
        "nonet_sandbox": "//tools/sandbox:nonet_sandbox",
        "noproc_sandbox": "//tools/sandbox:noproc_sandbox",
        "nonetproc_sandbox": "//tools/sandbox:nonetproc_sandbox",
    },
    labels = [
        "localonly",
        "manual",
    ],
    sandbox = False,
    deps = [
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
