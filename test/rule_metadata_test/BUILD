

sh_binary(
    name="content_checker",
    main="test_file_content.sh",
)

# TODO(jpoole): write more of these and move this to somewhere more common. Right now this is pretty specific to this
# test case. We might want to do more tests that just checking plz-out e.g. testing the output of plz query print.
def please_repo_e2e_test(
    name: str,
    plz_command: str,
    repo: str,
    deps: list=[],
    expected_output: dict,
):
    test_cmd = " && ".join(
        ["cd $DATA_REPO", f"$TMP_DIR/$DATA_PLEASE {plz_command}"] +
        [f"$TMP_DIR/$DATA_CONTENT_CHECKER '{o}' '{c}'" for o, c in expected_output.items()])
    return build_rule(
        name = name,
        test_cmd = test_cmd,
        deps = deps,
        data = {
            "REPO": [repo],
            "PLEASE": ["//src:please"],
            "CONTENT_CHECKER": [":content_checker"],
        },
        test = True,
        no_test_output = True,
    )



please_repo_e2e_test(
    name="rule_metadata_test",
    plz_command="build //consumer_package:wibble",
    repo="test_repo",
    expected_output={
        "plz-out/gen/consumer_package/wibble.txt": "wibble wibble wibble",
    }
)