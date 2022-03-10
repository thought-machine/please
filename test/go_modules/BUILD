subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "go_install_test",
    labels = [
        "no-musl",
    ],
    plz_command = "plz -o go.gotool:$TOOLS_GO test",
    repo = "test_repo",
    tools = {"GO": "//third_party/go:toolchain|go"},
)

please_repo_e2e_test(
    name = "pkg_config_test",
    labels = ["manual"],
    plz_command = "plz build -o go.gotool:$TOOLS_GO //third_party/go:all",
    repo = "pkg_config_repo",
    tools = {"go": ["//third_party/go:toolchain|go"]},
)

please_repo_e2e_test(
    name = "go_install_build",
    labels = [
        "no-musl",
    ],
    plz_command = "plz build -o go.gotool:$TOOLS_GO //third_party/...",
    repo = "test_repo",
    tools = {"go": ["//third_party/go:toolchain|go"]},
)

target = "darwin_amd64" if CONFIG.OS == "linux" else "linux_amd64"

please_repo_e2e_test(
    name = "cross_compile_test",
    labels = [
        "no_cirrus",
        "no-musl",
    ],
    plz_command = f"plz -o go.gotool:'//tools/go:toolchain|go' build --arch {target} //src/binary",
    repo = "test_repo",
)
