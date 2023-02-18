go_library(
    name = "asm",
    srcs = ["asm.go"],
    asm_srcs = ["add.s"],
    deps = ["//test/go_rules/asm/golib"],
)

go_test(
    name = "asm_test",
    srcs = ["asm_test.go"],
    external = True,
    labels = ["asm"],
    deps = [
        ":asm",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
