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
    # Doesn't seem to work on CircleCI - the client is inexplicably unable to connect
    # to the server, even though that's running. Maybe something related to its network config?
    labels = ["no_circleci"],
    deps = [
        ":follow",
        "//src/cli",
        "//src/core",
        "//third_party/go:testify",
    ],
)
