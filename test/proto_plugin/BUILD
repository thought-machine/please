subinclude("//test/build_defs:build_defs")

please_repo_e2e_test(
    name = "proto_rules_test",
    plz_command = "plz test",
    repo = "test_repo",
    # the protoc releases page doesn't support these platforms
    labels = ["no_cirrus", "no-musl"],
)