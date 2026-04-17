subinclude("//test/build_defs:build_defs")

filegroup(
    name = "export_e2e_test_build_def",
    srcs = ["please_export_e2e_test.build_defs"],
    visibility = ["//test/export/..."],
)

# Generic catch-all test on internal repo.
plz_e2e_test(
    name = "export_src_please_test",
    cmd = "plz export --output plz-out/plzexport //src/core && plz --repo_root=$(plz query reporoot)/plz-out/plzexport build //src/core",
)
