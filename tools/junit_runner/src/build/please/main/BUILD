java_library(
    name = "junit_runner",
    srcs = glob(["*.java"]),
    exported_deps = [
        "//third_party/java:hamcrest",
        "//third_party/java:junit",
        "//third_party/java/jacoco",
    ],
    visibility = ["//tools/junit_runner/..."],
    deps = [
        "//tools/junit_runner/src/build/please/common/report",
        "//tools/junit_runner/src/build/please/common/source",
        "//tools/junit_runner/src/build/please/common/test",
        "//tools/junit_runner/src/build/please/cover/report",
        "//tools/junit_runner/src/build/please/cover/result",
        "//tools/junit_runner/src/build/please/cover/runner",
        "//tools/junit_runner/src/build/please/test/report",
        "//tools/junit_runner/src/build/please/test/result",
        "//tools/junit_runner/src/build/please/test/runner",
    ],
)
