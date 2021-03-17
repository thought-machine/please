subinclude("//test/build_defs")

proto_library(
    name = "root_dir",
    srcs = [
        "other_test.proto",
        "test.proto",
    ],
    languages = ["go"],
    root_dir = "test/proto_rules",
)

plz_e2e_test(
    name = "root_dir_test",
    cmd = "plz build //test/proto_rules/root_dir:_root_dir#protoc_go",
    expect_output_contains = "plz-out/gen/test/proto_rules/root_dir/test.pb.go",
)
