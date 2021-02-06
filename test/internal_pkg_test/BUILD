subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "internal_pkg_test",
    expected_output = {
        "out.txt": "jarcat: //_please:tools|jarcat",
    },
    labels = ["manual"],
    plz_command = "plz build //src:unzip && plz query print -f tools //src:unzip > out.txt",
    repo = "test_repo",
    set_tool_config = False,
)
