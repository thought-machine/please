subinclude("//build_defs:plz_e2e_test")

go_binary(
    name = "go_provider",
    srcs = ["main.go"],
    deps = [
        "//third_party/go:logging",
    ],
)

plz_e2e_test(
    name = "provider_test",
    cmd = "plz test //test/go_provider/example:example_test",
)
