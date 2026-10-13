subinclude("//test/build_defs")

# A target's build command refers to the outputs of a target in its data that has output directories.
# Those outputs can't be known until it's built, so it has to be built before the target.
please_repo_e2e_test(
    name = "data_output_dirs_test",
    timeout = 60,
    expected_output = {
        "plz-out/gen/a/gen.txt": "a/x.txt",
    },
    plz_command = "plz build //a:gen",
    repo = "test_repo",
    # Otherwise a bad result could be retrieved from the cache instead of testing the ordering.
    shared_cache = False,
)
