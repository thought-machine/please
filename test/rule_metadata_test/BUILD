subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "rule_metadata_test",
    expected_output = {
        "plz-out/gen/consumer_package/wibble.txt": "wibble wibble wibble",
    },
    plz_command = "build //consumer_package:wibble",
    repo = "test_repo",
)
