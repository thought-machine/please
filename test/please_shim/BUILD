subinclude("//test/build_defs:build_defs")

please_repo_e2e_test(
    name = "install",
    expect_output_contains = {
        # Version installed must match the one in the .plzconfig file.
        "plz_version.txt": "Please version 17.11.0",
    },
    plz_command = " && ".join([
        # Install Please since it won't find it. Please location
        # has been set to `./please` relative to the repo root.
        "$TOOLS_PLEASE_SHIM --version > plz_version.txt",
        # Guarantee it was installed in the right place.
        "[ -e $($TOOLS_PLEASE_SHIM query config please.location) ]",
    ]),
    repo = "test_repo",
    tools = {
        "please_shim": "//tools/please_shim",
    },
)

please_repo_e2e_test(
    name = "no_binary_no_completion",
    expected_output = {
        # No completions.
        "completions.txt": "",
    },
    plz_command = " && ".join([
        # Try argument completion for non-existant binary.
        "GO_FLAGS_COMPLETION=1 $TOOLS_PLEASE_SHIM 2>&1 > completions.txt",
        # Guarantee that it didn't install Please.
        "! [ -e please/please ]",
    ]),
    repo = "test_repo",
    tools = {
        "please_shim": "//tools/please_shim",
    },
)

please_repo_e2e_test(
    name = "update",
    expect_output_contains = {
        # Version installed must match the one in the .plzconfig file.
        "install_version.txt": "Please version 17.11.0",
        # Update version.
        "update_version.txt": "Please version 17.10.0",
    },
    plz_command = " && ".join([
        # Install binary first.
        "$TOOLS_PLEASE_SHIM --version > install_version.txt",
        # Update version.
        "$TOOLS_PLEASE_SHIM --profile=update_version --version > update_version.txt",
    ]),
    repo = "test_repo",
    tools = {
        "please_shim": "//tools/please_shim",
    },
)

please_repo_e2e_test(
    name = "completion",
    expect_output_contains = {
        "completions.txt": "build",
        "completions.txt": "test",
        "completions.txt": "cover",
    },
    plz_command = " && ".join([
        # Install binary first.
        "$TOOLS_PLEASE_SHIM --version",
        # Trigger completion.
        "GO_FLAGS_COMPLETION=1 $TOOLS_PLEASE_SHIM > completions.txt",
    ]),
    repo = "test_repo",
    tools = {
        "please_shim": "//tools/please_shim",
    },
)
