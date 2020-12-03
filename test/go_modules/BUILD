subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "go_modules_test",
    plz_command = "plz test",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "go_modules_test_please_go_install",
    plz_command = "plz test -o featureflags.PleaseGoInstall:true",
    repo = "test_repo",
)
