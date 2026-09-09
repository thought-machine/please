subinclude("//test/export:export_e2e_test_build_def")

# This test validates the trimming of if/elif/else blocks during export.
# Specifically, it tests that when exporting a target defined inside an 'elif' block,
# the other unused 'if' and 'else' blocks are correctly trimmed and replaced with 'pass'.
please_export_e2e_test(
    name = "export_elif",
    enforce_different_repos = False,
    export_targets = ["//:b"],
)
