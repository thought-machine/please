package(javac_tool = "javac")

java_library(
    name = "lib",
    srcs = ["Lib.java"],
    resources = ["data.txt"],
)

java_test(
    name = "Test",
    srcs = ["Test.java"],
    deps = [
        ":lib",
        "//third_party/java:junit",
    ],
)
