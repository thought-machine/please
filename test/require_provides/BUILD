subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "require_provide_test",
    expected_output = {
        "plz-out/gen/foo.ls": "foo.txt",  # Nothing provided for foo, so we get the filegroup rule
        "plz-out/gen/bar.ls": "bar.txt",  # bar matches a provide so we get the bar file instead
        "plz-out/gen/baz.ls": "",  # baz matches a provide on an empty list so we get nothing
        "plz-out/gen/nothing.ls": "foo.txt",  # Similar to foo, this matches the main rule
    },
    plz_command = "plz build :all",
    repo = "test_repo",
)
