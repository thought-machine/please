subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "toolchain_test",
    expected_output = {
        "out.txt": "wibble wibble wibble",
    },
    labels = ["no-musl"],
    plz_command = "plz test //src/foo:foo_test && plz run //src:main > out.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "freebsd_arm_toolchain_test",
    expected_output = {
        "out.txt": "plz-out/bin/freebsd_arm64/src/main: ELF 64-bit LSB executable, ARM aarch64, version 1 (FreeBSD), statically linked, stripped",
    },
    labels = ["no-musl"],
    plz_command = "plz build //src:main --arch freebsd_arm64 && file plz-out/bin/freebsd_arm64/src/main > out.txt",
    repo = "test_repo",
)
