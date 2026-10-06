subinclude("//test/build_defs")

# A target's build command refers to the outputs of a filegroup in its data, whose source is in a
# package that's slow to parse. The filegroup's outputs can't be calculated until that source has
# been parsed, even though it doesn't need to be built before the target.
please_repo_e2e_test(
    name = "data_filegroup_outputs_test",
    timeout = 60,
    expected_output = {
        "plz-out/gen/a/gen.txt": "a/x.txt",
    },
    plz_command = "plz build //a:gen",
    repo = "test_repo",
    # The slow parse is the point of the test, so it mustn't be retrieved from the cache.
    shared_cache = False,
)
