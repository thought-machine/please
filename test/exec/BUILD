subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "exec_test",
    expected_output = {"plz-out/exec_test/out.txt": "test"},
    plz_command = "plz exec --out out.txt --output_path plz-out/exec_test //:dummy -- \"echo test > out.txt\"",
    repo = "test_repo",
)
