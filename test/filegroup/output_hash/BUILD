subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "output_hash_test",
    plz_command = "plz build //srcs:filegroup",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "update_dep_hash_test",
    plz_command = "plz build //srcs:filegroup && rm srcs/file2 && touch srcs/file2 && plz build //srcs:filegroup",
    repo = "test_repo",
    expected_failure=True,
)

please_repo_e2e_test(
    name = "update_src_hash_test",
    plz_command = "plz build //srcs:filegroup && rm srcs/file1 && touch srcs/file1 && plz build //srcs:filegroup",
    repo = "test_repo",
    expected_failure=True,
)