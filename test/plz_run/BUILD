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
    name = "env_test_in_tmp_dir",
    expected_output = {
        "wibble.txt": "src/wibble.txt src/wibble.sh",
    },
    plz_command = "plz run //src:env_printer --in_tmp_dir > wibble.txt",
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

# Test add_data functionality with a dummy file
please_repo_e2e_test(
    name = "_add_data_file_test",
    plz_command = "plz test //src:_add_data_file",
    repo = "test_repo",
)

# Test add_data functionality with a file and a build target
please_repo_e2e_test(
    name = "_add_data_target_test",
    plz_command = "plz test //src:_add_data_file_and_target",
    repo = "test_repo",
)

# Test --wd flag changes directory and outputs there
please_repo_e2e_test(
    name = "test_wd_flag",
    expected_output = {
        "test_wd/guava.txt": "guava",
    },
    plz_command = "plz run --wd test_wd //src:create_file",
    repo = "test_repo",
)
