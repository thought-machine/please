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
