subinclude("//test/build_defs")

please_repo_e2e_test (
    name = "toolchain_test",
    plz_command = "plz test //src/foo:foo_test && plz run //src:main > out.txt",
    expected_output = {
        "out.txt": "wibble wibble wibble",
    },
    repo = "test_repo",
    #TODO(jpoole): Fix this on FreeBSD and re-enable this test of cirrus
    labels = ["no-musl", "no_cirrus"],
)
