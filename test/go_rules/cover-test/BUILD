# This deliberately has hyphens in the name to mimic #537
proto_library(
    name = "test-proto",
    srcs = ["test-proto.proto"],
)

go_test(
    name = "cover_test",
    srcs = ["cover_test.go"],
    deps = [
        ":test-proto",
        "//third_party/go:testify",
    ],
)
