go_test(
    name = "sandbox_test",
    srcs = ["sandbox_test.go"],
    data = {
        "please_sandbox": "//tools/sandbox:please_sandbox",
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
