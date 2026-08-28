subinclude("//test/export:export_e2e_test_build_def")

# This test validates that if a target is assigned to a variable,
# the variable assignment statement is correctly trimmed if the target
# referencing it is unused/not exported, and kept if the target is exported.
please_export_e2e_test(
    name = "target_variable_trimming",
    export_targets = ["//:used"],
)
