subinclude("//test/build_defs")

BUILD_FILE = """
python_binary(
name = "foo",
main = "main.py"
)
""".strip()

FORMATTED_BUILD_FILE = """
python_binary(
    name = "foo",
    main = "main.py",
)
""".strip()

please_repo_e2e_test(
    name = "outputs_formatted_build_file_to_stdout",
    expected_output = {
        "src/BUILD_FILE": BUILD_FILE,
        "output": FORMATTED_BUILD_FILE,
    },
    plz_command = "plz format src/BUILD_FILE > output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "fails_when_file_needs_formatting_in_quiet_mode",
    expected_failure = True,
    expected_output = {
        "src/BUILD_FILE": BUILD_FILE,
    },
    plz_command = "plz format --quiet src/BUILD_FILE",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "succeeds_when_file_doesnt_need_formatting_in_quiet_mode",
    expected_output = {
        "src/BUILD_FILE": BUILD_FILE,
    },
    plz_command = "plz format --quiet src/FORMATTED_BUILD_FILE",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "writes_formatted_file_back_in_write_mode",
    expected_output = {
        "src/BUILD_FILE": FORMATTED_BUILD_FILE,
    },
    plz_command = "plz format --write src/BUILD_FILE",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "fails_when_quiet_and_write_mode_given",
    expect_output_contains = {
        "output": "Can't use both --quiet and --write at the same time",
    },
    expected_failure = True,
    expected_output = {
        "src/BUILD_FILE": BUILD_FILE,
    },
    plz_command = "plz format --write --quiet src/FORMATTED_BUILD_FILE 2> output",
    repo = "test_repo",
)
