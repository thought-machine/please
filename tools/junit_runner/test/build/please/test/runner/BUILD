java_test(
    name = "please_test_runner_test",
    srcs = ["PleaseTestRunnerTest.java"],
    deps = [
        ":test_examples",
        ":test_runner",
        "//tools/junit_runner/src/build/please/test/result",
        "//tools/junit_runner/src/build/please/test/runner",
        "//third_party/java:junit",
    ],
)

java_library(
    name = "test_runner",
    srcs = [
        "AlwaysAcceptingPleaseTestRunner.java",
    ],
    deps = [
        "//tools/junit_runner/src/build/please/test/runner",
    ],
    test_only = True,
    visibility = [
        "//tools/junit_runner/test/build/please/cover/runner/...",
    ]
)

java_library(
    name = "test_examples",
    srcs = [
        "AssertionFailure.java",
        "CaptureOutput.java",
        "Error.java",
        "OnlySuccess.java",
        "Parameterized.java",
        "SkippedAndSuccess.java",
    ],
    deps = [
        "//tools/junit_runner/src/build/please/common/test",
        "//third_party/java:junit",
    ]
)
