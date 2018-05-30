java_test(
    name = "please_coverage_runner_test",
    srcs = [
        "PleaseCoverageRunnerTest.java",
    ],
    deps = [
        "//third_party/java:junit",
        "//third_party/java/jacoco",
        "//tools/junit_runner/src/build/please/cover/result",
        "//tools/junit_runner/src/build/please/cover/runner",
        "//tools/junit_runner/src/build/please/test/result",
        "//tools/junit_runner/src/build/please/test/runner",
        "//tools/junit_runner/test/build/please/cover/runner/testdata",
        "//tools/junit_runner/test/build/please/test/runner:test_runner",
    ],
)

java_test(
    name = "instrumenting_class_loader_test",
    srcs = [
        "InstrumentingClassLoaderTest.java",
    ],
    deps = [
        "//third_party/java:junit",
        "//third_party/java/jacoco",
        "//tools/junit_runner/src/build/please/cover/runner",
        "//tools/junit_runner/test/build/please/cover/runner/testdata",
    ],
)
