subinclude("//test/build_defs:build_defs")

OUTPUT_DIR = "plz-out/plzexport"

EXPECTED_FILES = "expected"

# Test outputs export.
please_repo_e2e_test(
    name = "export_outputs",
    data = {
        "EXPECTED": [EXPECTED_FILES],
    },
    defer_cmd = [
        # Golden-Master validation with expected files. Test data is added to HOME dir.
        f'diff -ru "{OUTPUT_DIR}" "$HOME/$DATA_EXPECTED"',
    ],
    plz_command = f'plz export outputs --output "{OUTPUT_DIR}" //:out_target',
    repo = "source_repo",
)
