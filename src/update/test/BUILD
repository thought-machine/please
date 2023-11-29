subinclude("//build_defs:version")

go_binary(
    name = "please",
    srcs = ["test_main.go"],
    definitions = {"main.pleaseVersion": VERSION},
    test_only = True,
    visibility = ["//src/update:all"],
)
