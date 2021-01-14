subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "go_install_test",
    plz_command = "plz test",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "please_go_install_test",
    plz_command = "plz -o featureflags.PleaseGoInstall:true test",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "please_go_install_cross_compile_test",
    plz_command = "plz -o featureflags.PleaseGoInstall:true -o go.gotool:'//tools/go:toolchain|go' build --arch linux_arm64 //third_party/go:all",
    repo = "test_repo",
)