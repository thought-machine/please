subinclude("//test/build_defs")

# We need to pass in `--share_mount` due to permission privileges in CircleCI

please_repo_e2e_test(
    name = "binary_target",
    expected_output = {
        "output": "hello world",
    },
    plz_command = "plz exec --share_mount //:hello_world >output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "network_unshared",
    expected_failure = True,
    plz_command = "plz exec --share_mount //:network_test",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "network_shared",
    plz_command = "plz exec --share_mount --share_network //:network_test",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "mount_shared",
    expect_output_contains = {
        "output": "test/plz_exec/test_repo/plz-out/exec",
    },
    plz_command = "plz exec --share_mount //:hello_world -- pwd >output",
    repo = "test_repo",
)
