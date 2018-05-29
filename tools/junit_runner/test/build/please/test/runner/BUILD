java_test(
    name = "please_test_runner_test",
    srcs = ["PleaseTestRunnerTest.java"],
    deps = [
        ":test_runner",
        "//tools/junit_runner/src/build/please/test/result",
        "//tools/junit_runner/src/build/please/test/runner",
        "//tools/junit_runner/test/build/please/test/runner/testdata",
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
