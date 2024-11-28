subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "subinclude_cycle_test",
    plz_command = "for i in `seq 1 200`; do plz build -p -v 2 //...; done",
    repo = "test_repo",
)
