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
