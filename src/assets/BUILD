go_library(
    name = "assets",
    srcs = ["assets.go"],
    pgo_file = "//:pgo",
    resources = [":resources"],
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
