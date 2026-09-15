subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "cyclic_dependency_test",
    expect_output_contains = {
        "output.txt": "Dependency cycle found",
    },
    expected_failure = True,
    plz_command = "plz test //cycle:all > output.txt 2>&1",
    repo = "test_repo",
)
