package(
    GRPC_GO_PLUGIN = "//third_party/go:protoc-gen-go-grpc",
    PROTOC_GO_PLUGIN = ":protoc-gen-go",
)

# This is different from //third_party/go:protoc-gen-go which is the old unified grpc and proto plugin
go_module(
    name = "protoc-gen-go",
    binary = True,
    install = ["cmd/protoc-gen-go"],
    module = "google.golang.org/protobuf",
    version = "v1.25.0",
    deps = [
        "//third_party/go:cmp",
    ],
)

grpc_library(
    name = "proto",
    srcs = ["service.proto"],
    visibility = ["//test/proto_rules/go_grpc_ff/..."],
)
