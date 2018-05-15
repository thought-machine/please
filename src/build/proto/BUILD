# This is a dependency of our in-repo compiler, so we must use the system compiler.
package(javac_tool = "javac")

proto_library(
    name = "worker",
    srcs = ["worker.proto"],
    languages = [
        "go",
        "java",
    ],
    visibility = ["PUBLIC"],
)
