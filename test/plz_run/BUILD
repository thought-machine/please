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
    expect_output_contains = {
        # Bit of a hack... dicts can only have the file once so we copy it to check it contains both.
        "wibble.txt": "wibble",
        "wobble.txt": "wobble",
    },
    plz_command = "echo //src:wibble //src:wobble | plz run parallel - > wibble.txt && cp wibble.txt wobble.txt",
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
        "test_wd/fruit.txt": "guava",
    },
    plz_command = "mkdir test_wd; plz run --wd test_wd //src:write_guava_to_file",
    repo = "test_repo",
)

# Test --wd works with relative paths
please_repo_e2e_test(
    name = "test_wd_relative_paths",
    expected_output = {
        "path/to/some/dir/fruit.txt": "guava",
    },
    plz_command = "mkdir -p path/to/some/dir/ && cd path/to && plz run --wd ./some/dir //src:write_guava_to_file && cd -",
    repo = "test_repo",
)
