subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "exec_test",
    plz_command = "plz exec --out out.txt --output_path plz-out/exec_test //:dummy -- \"echo test > out.txt\"",
    expected_output = {"plz-out/exec_test/out.txt": "test"},
    repo = "test_repo",
)