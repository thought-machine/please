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

target = "darwin_amd64" if CONFIG.OS == "linux_amd64" else "linux_amd64"

please_repo_e2e_test(
    name = "please_go_install_cross_compile_test",
    labels = [
        "no_cirrus",
        "no-musl",
    ],
    plz_command = f"plz -o featureflags.PleaseGoInstall:true -o go.cgoenabled:1 -o go.gotool:'//tools/go:toolchain|go' build --arch {target} //src/binary",
    repo = "test_repo",
)
