# Tests on the proto rules.

proto_library(
    name = "root_dir",
    srcs = ["test.proto", "other_test.proto"],
    languages = ["go"],
    deps = [":protoc"],
    root_dir = "test\/proto_rules",
)

# Required for wkt
protoc_binary(
    name = "protoc",
    version = "3.15.1",
    visibility = ["PUBLIC"],
)
