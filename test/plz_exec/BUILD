subinclude("//test/build_defs")

# Only work with linux namespaces
if CONFIG.OS == "linux":
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
        expect_output_contains = {
            "output": "Dial error: ",
        },
        plz_command = "plz exec --share_mount //:network_test 2>output",
        repo = "test_repo",
    )

    please_repo_e2e_test(
        name = "network_shared",
        expect_output_doesnt_contain = {
            "output": "Dial error: ",
        },
        plz_command = "plz exec --share_mount --share_network //:network_test 2>output",
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
