subinclude("//test/build_defs:build_defs")

filegroup(
    name = "export_e2e_test_build_def",
    srcs = ["please_export_e2e_test.build_defs"],
    visibility = ["//test/export/..."],
)

# Generic catch-all test on internal repo.
plz_e2e_test(
    name = "export_src_please_test",
    cmd = f'plz export --output "$PLZ_EXPORT_DIR" //src/core:core && plz --repo_root="$PLZ_EXPORT_DIR" build //src/core:core',
    pre_cmd = f'export PLZ_EXPORT_DIR="$(mktemp -d)"',
)
