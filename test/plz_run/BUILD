subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "env_test",
    expected_output = {
        "wibble.txt": "src/wibble.txt plz-out/bin/src/wibble.sh",
    },
    plz_command = "plz run //src:env_printer --env > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "std_in_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = "echo //src:wibble | plz run - > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "std_in_parallel_test",
    expected_output = {
        "wibble.txt": "wibble\nwobble",
    },
    plz_command = "echo //src:wibble //src:wobble | plz run parallel - > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "std_in_sequential_test",
    expected_output = {
        "wibble.txt": "wibble\nwobble",
    },
    plz_command = "echo //src:wibble //src:wobble | plz run sequential - > wibble.txt",
    repo = "test_repo",
)
