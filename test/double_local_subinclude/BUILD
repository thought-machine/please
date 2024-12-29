subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "double_local_subinclude_test",
    # This isn't 100% deterministic but 20 clean runs should be reasonably sufficient
    plz_command = "for i in `seq 1 20`; do plz build -p -v 2 //...; done",
    repo = "test_repo",
)
