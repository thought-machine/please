subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "srcs_test",
    expected_output = {
        "plz-out/gen/with_srcs.txt": "a.txt\nb.txt",
    },
    plz_command = "plz build //:with_srcs",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "src_test",
    expected_output = {
        "plz-out/gen/with_one_src.txt": "a.txt",
    },
    plz_command = "plz build //:with_one_src",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "named_srcs_test",
    expected_output = {
        "plz-out/gen/with_named_srcs.txt": "a.txt\nb.txt",
    },
    plz_command = "plz build //:with_named_srcs",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "flag_not_set_test",
    expected_failure = True,
    plz_command = "plz build //:with_flag_not_set",
    repo = "test_repo",
)
