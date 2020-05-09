go_library(
    name = "follow",
    srcs = [
        "grpc_client.go",
        "grpc_server.go",
        "marshalling.go",
        "resources.go",
    ],
    visibility = ["PUBLIC"],
    deps = [
        "//src/core",
        "//src/follow/proto:build_event",
        "//src/output",
        "//third_party/go:grpc",
        "//third_party/go:logging",
        "//third_party/go:psutil",
    ],
)

go_test(
    name = "grpc_test",
    srcs = ["grpc_test.go"],
    labels = ["manual"],  # Having many issues with this test. Feature is soon to be removed.
    deps = [
        ":follow",
        "//src/cli",
        "//src/core",
        "//third_party/go:testify",
    ],
)
