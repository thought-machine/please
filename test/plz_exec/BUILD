subinclude("//test/build_defs")

# Only work with linux namespaces. These are manual because you can't do elevated namespace operations on the docker
# runner in circle.
if CONFIG.OS == "linux":
    please_repo_e2e_test(
        name = "network_unshared",
        expect_output_contains = {
            "output": "Dial error: ",
        },
        expected_failure = True,
        labels = ["manual"],
        plz_command = "plz exec --profile sandbox --share_mount //:network_test 2>output",
        repo = "test_repo",
    )

    please_repo_e2e_test(
        name = "network_shared",
        expect_output_doesnt_contain = {
            "output": "Dial error: ",
        },
        labels = ["manual"],
        plz_command = "plz exec --profile sandbox --share_mount --share_network //:network_test 2>output",
        repo = "test_repo",
    )

    please_repo_e2e_test(
        name = "mount_shared",
        expect_output_contains = {
            "output": "test/plz_exec/test_repo/plz-out/exec",
        },
        labels = ["manual"],
        plz_command = "plz exec --profile sandbox --share_mount //:hello_world -- pwd >output",
        repo = "test_repo",
    )

please_repo_e2e_test(
    name = "binary_target",
    expected_output = {
        "output": "hello world",
    },
    plz_command = "plz exec //:hello_world >output",
    repo = "test_repo",
)
