go_test(
    name = "sandbox_test",
    srcs = ["sandbox_test.go"],
    data = {
        "please_sandbox": "//tools/sandbox:please_sandbox",
    },
    labels = [
        # These tests are executed explicitly via machine executors in .circleci/config.yml
        # For local testing execute in a container with equivalent configuration.
        "manual",
    ],
    sandbox = False,
    deps = [
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
