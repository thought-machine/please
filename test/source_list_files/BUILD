subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "srcs_test",
    plz_command = "plz build //:with_srcs",
    expected_output = {
      "plz-out/gen/with_srcs.txt": "a.txt\nb.txt",
    },
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "src_test",
    plz_command = "plz build //:with_one_src",
    expected_output = {
      "plz-out/gen/with_one_src.txt": "a.txt",
    },
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "named_srcs_test",
    plz_command = "plz build //:with_named_srcs",
    expected_output = {
      "plz-out/gen/with_named_srcs.txt": "a.txt\nb.txt",
    },
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "flag_not_set_test",
    plz_command = "plz build //:with_flag_not_set",
    repo = "test_repo",
    expected_failure = True,
)
