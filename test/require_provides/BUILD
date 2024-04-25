subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "require_provide_test",
    expected_output = {
        "plz-out/gen/foo.ls": "foo.txt", # Nothing provided for foo, so we get the filegroup rule
        "plz-out/gen/bar.ls": "bar.txt", # bar matches a provide so we get the bar file instead
        "plz-out/gen/baz.ls": "", # baz matches a provide on an empty list so we get nothing
        "plz-out/gen/nothing.ls": "foo.txt", # Similar to foo, this matches the main rule
    },
    plz_command = "plz build :all",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "require_provide_default_ff_on_test",
    expected_output = {
        "plz-out/gen/default_foo.ls": "foo.txt", # Foo matches so we get foo
        "plz-out/gen/default_bar.ls": "bar.txt", # bar doesn't match so we get the default
    },
    plz_command = "plz build -o featureflags.FFDefaultProvides:true :all",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "require_provide_default_ff_off_test",
    expected_output = {
        "plz-out/gen/default_foo.ls": "foo.txt", # Foo matches so we get foo
        "plz-out/gen/default_bar.ls": "", # bar doesn't match so we get nothing
    },
    plz_command = "plz build :all",
    repo = "test_repo",
)