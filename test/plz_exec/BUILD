subinclude("//test/build_defs")

# We need to pass in `--share_mount` due to permission privileges in CircleCI

please_repo_e2e_test(
    name = "binary_target",
    expected_output = {
        "output": "hello world",
    },
    plz_command = "plz exec --share_mount //:hello_world > output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "network_unshared",
    exit_code = 6,
    expect_output_contains = {
        "output": "Could not resolve host",
    },
    plz_command = "plz exec --share_mount //:hello_world -- curl google.com 2>output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "network_shared",
    expect_output_doesnt_contain = {
        "output": "Could not resolve host",
    },
    plz_command = "plz exec --share_network --share_mount //:hello_world -- curl google.com 2>output",
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
