subinclude("//test/build_defs")


success_out = '{"1": [//test/sub1:sub1|1], "2": [//test/sub1:sub1|2], "3": [//test/sub2:sub2|3], "4": [//test/sub2:sub2|4]}'
please_repo_e2e_test(
    name = "targetmeta_success",
    plz_command = "plz build //test:success",
    expected_output = { "plz-out/gen/test/success": success_out },
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "targetmeta_unknown_target",
    plz_command = "plz build",
    expect_output_contains = {
        "plz-out/log/build.log": "depends on //inexistent_pkg:inexistent_pkg, but the directory inexistent_pkg doesn't exist"
    },
    expected_failure = True,
    repo = "error/unknown_target",
)

please_repo_e2e_test(
    name = "targetmeta_unknown_local_target",
    plz_command = "plz build",
    expect_output_contains = {
        "plz-out/log/build.log": "CRITICAL: Target //:local not found in build graph"
    },
    expected_failure = True,
    repo = "error/unknown_local_target",
)


# This test can output one of two line depending on the order files are finished parsing
out1 = "Parsed build file extern/BUILD_FILE but it doesn't contain target b"
out2 = "CRITICAL: Target //extern:b (referenced by //:all) doesn't exist"
log_file = "plz-out/log/build.log"
test_cmd = f'_STR="$(cat {log_file})" _SUBSTR1="{out1}" _SUBSTR2="{out2}"'
test_cmd += ' && if [ "${_STR##*$_SUBSTR1*}" ] && [ "${_STR##*$_SUBSTR2*}" ]; then echo "$_STR"; exit 1; fi'

please_repo_e2e_test(
    name = "targetmeta_unknown_extern_target",
    plz_command = f"plz build; [ ! $? -eq 0 ] && {test_cmd}",
    repo = "error/unknown_extern_target",
)

please_repo_e2e_test(
    name = "targetmeta_non_visible_target",
    plz_command = "plz build",
    expect_output_contains = {
        "plz-out/log/build.log": "Target //extern:a isn't visible to"
    },
    expected_failure = True,
    repo = "error/non_visible_target",
)
