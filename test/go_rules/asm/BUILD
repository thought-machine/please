go_library(
    name = "asm",
    srcs = ["asm.go"],
    asm_srcs = ["add.s"],
)

go_test(
    name = "asm_test",
    srcs = ["asm_test.go"],
    external = True,
    labels = ["asm"],
    deps = [
        ":asm",
        "//third_party/go:testify",
    ],
)
