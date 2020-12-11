subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "preload_subinc_test",
    expected_output = {
        "out.txt": "//src:k8s",
    },
    plz_command = "plz query alltargets //src:k8s > out.txt",
    repo = "test_repo",
)
