subinclude("///proto//build_defs:proto")

proto_library(
    name = "proto_only",
    srcs = ["service.proto"],
    visibility = ["PUBLIC"],
)

grpc_library(
    name = "proto",
    srcs = ["service.proto"],
    visibility = ["PUBLIC"],
)