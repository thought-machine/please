cobol_library(
    name = "lib",
    srcs = ["HELLO.COB"],
)

cobol_binary(
    name = "bin",
    srcs = ["MAIN.COB"],
    deps = [":lib"],
)
