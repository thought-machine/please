subinclude("//test/build_defs:build_defs")

please_repo_e2e_test(
    name = "proto_rules_test",
    # the protoc releases page doesn't support these platforms
    labels = [
        "no_cirrus",
        "no-musl",
        "manual",
    ],
    plz_command = "plz -o plugin.go.gotool:$TOOLS_GO test",
    repo = "test_repo",
    tools = {
        "go": [CONFIG.GO_TOOL],
    },
)
