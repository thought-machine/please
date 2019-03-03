subinclude("//build_defs:version")

go_binary(
    name = "please",
    srcs = ["test_main.go"],
    deps = ["//src/core"],
    visibility = ["//src/update:all"],
    definitions = {"main.PleaseVersion": VERSION},
)
