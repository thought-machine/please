subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "exec_test",
    expected_output = {"plz-out/exec_test/out.txt": "test"},
    plz_command = "plz exec --out out.txt --output_path plz-out/exec_test //:dummy -- \"echo test > out.txt\"",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "exec_sequential_test",
    plz_command = "plz exec sequential //:success //:success2 hello",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "exec_sequential_failure_test",
    expected_failure = True,
    plz_command = "plz exec sequential //:failure",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "exec_parallel_test",
    plz_command = "plz exec parallel //:success //:success2 hello",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "exec_parallel_failure_test",
    expected_failure = True,
    plz_command = "plz exec parallel //:failure",
    repo = "test_repo",
)
