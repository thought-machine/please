go_library(
    name = "remote",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//src/build",
        "//src/core",
        "//src/fs",
        "//src/process",
        "//third_party/go:errgroup",
        "//third_party/go:genproto_api",
        "//third_party/go:genproto_rpc",
        "//third_party/go:grpc",
        "//third_party/go:grpc-middleware",
        "//third_party/go:logging",
        "//third_party/go:protobuf",
        "//third_party/go:remote-apis",
        "//third_party/go:remote-apis-sdks",
        "//third_party/go:uuid",
    ],
)

go_test(
    name = "remote_test",
    srcs = [
        "impl_test.go",
        "remote_test.go",
    ],
    data = ["test_data"],
    # TODO(#1412): find out why this flakes on circle
    flaky = True,
    deps = [
        ":remote",
        "//src/core",
        "//third_party/go:genproto_api",
        "//third_party/go:genproto_rpc",
        "//third_party/go:grpc",
        "//third_party/go:protobuf",
        "//third_party/go:remote-apis",
        "//third_party/go:sri",
        "//third_party/go:testify",
    ],
)
