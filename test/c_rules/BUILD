subinclude("///cc//build_defs:c")

c_binary(
    name = "select",
    srcs = ["select.c"],
)

sh_test(
    name = "select_test",
    src = "select_test.sh",
    data = [":select"],
)
