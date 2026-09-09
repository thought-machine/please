subinclude("//test/export:export_e2e_test_build_def")

# Export a target inside an if block and validate that the else block is correctly trimmed (and replaced with pass).
please_export_e2e_test(
    name = "export_if",
    export_targets = ["//:a"],
)
