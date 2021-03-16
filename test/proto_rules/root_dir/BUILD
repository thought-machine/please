subinclude("//test/build_defs")

proto_library(
    name = "root_dir",
    srcs = [
        "other_test.proto",
        "test.proto",
    ],
    languages = ["go"],
    root_dir = "test\\/proto_rules",
)

plz_e2e_test(
    name = "query_somepath_reverse_test",
    cmd = "plz build //test/proto_rules/root_dir/...",
)
