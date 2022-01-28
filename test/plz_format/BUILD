subinclude("//test/build_defs")

FORMATTED_BUILD_FILE = """
python_binary(
    name = "foo",
    main = "main.py",
)
""".strip()

please_repo_e2e_test(
    name = "outputs_formatted_build_file_to_stdout",
    expected_output = {
        "output": FORMATTED_BUILD_FILE,
    },
    plz_command = "plz format src/BUILD_FILE > output",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "fails_when_file_needs_formatting_in_quiet_mode",
    plz_command = "plz format --quiet src/BUILD_FILE",
    expected_failure = True,
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "succeeds_when_file_doesnt_need_formatting_in_quiet_mode",
    plz_command = "plz format --quiet src/FORMATTED_BUILD_FILE",
    repo = "test_repo",
)
