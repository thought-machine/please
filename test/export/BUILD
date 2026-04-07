subinclude("//test/build_defs:build_defs")

filegroup(
    name = "export_e2e_test_build_def",
    srcs = ["please_export_e2e_test.build_defs"],
    visibility = ["//test/export/..."],
)

# Test outputs export.
please_repo_e2e_test(
    name = "export_outputs",
    defer_cmd = [
        'test -f "plz-out/plzexport/some_output.txt"',
        'test ! -f "plz-out/plzexport/BUILD_FILE"',
    ],
    plz_command = "plz export outputs --output plz-out/plzexport //:out_target",
    repo = "test_outputs",
)

# Test --notrim flag.
please_repo_e2e_test(
    name = "export_notrim",
    defer_cmd = [
        'test -f "plz-out/plzexport/file2.txt"',
        'test -f "plz-out/plzexport/BUILD_FILE"',
        'grep -q \'name = "unrelated"\' "plz-out/plzexport/BUILD_FILE"',
    ],
    plz_command = "plz export --notrim --output plz-out/plzexport //:target1",
    repo = "test_notrim",
)

# Generic catch-all test on internal repo.
plz_e2e_test(
    name = "export_src_please_test",
    cmd = "plz export --output plz-out/plzexport //src/core && plz --repo_root=$(plz query reporoot)/plz-out/plzexport build //src/core",
)
