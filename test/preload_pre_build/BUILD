subinclude("//test/build_defs")

# A preloaded subinclude that needs a target with a pre-build function. Running that function parses
# a package in the same repo, which must not wait on the preloads it's part of resolving.
please_repo_e2e_test(
    name = "preload_pre_build_test",
    expected_output = {
        "out.txt": "//src:src",
    },
    plz_command = "plz build //src:src && plz query alltargets //src:all > out.txt",
    repo = "test_repo",
    timeout = 120,
)
