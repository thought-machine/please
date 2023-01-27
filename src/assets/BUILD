go_library(
    name = "assets",
    srcs = ["assets.go"],
    resources = glob([
        "pleasew",
        "plz_complete.sh",
    ]),
    visibility = ["//src/..."],
)

go_test(
    name = "assets_test",
    srcs = ["assets_test.go"],
    # Has to be external (or provide the same resources as above)
    external = True,
    deps = [":assets"],
)

filegroup(
    name = "resources",
    srcs = [
        "//:pleasew",
        "//tools/misc:completion_script",
    ],
)
