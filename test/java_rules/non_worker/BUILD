subinclude("///java//build_defs:java")

package(
    java_toolchain = "",
    javac_tool = "javac",
)

java_library(
    name = "lib",
    srcs = ["Lib.java"],
    resources = ["data.txt"],
)

java_test(
    name = "non_worker_test",
    srcs = ["NonWorkerTest.java"],
    resources = ["data2.txt"],
    deps = [
        ":lib",
        "//third_party/java:junit",
    ],
)
