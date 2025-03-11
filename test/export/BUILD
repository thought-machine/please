subinclude("//test/build_defs")

plz_e2e_test(
    name = "export_src_please_test",
    cmd = "plz export --output plz-out/plzexport //src/core && plz --repo_root=$(plz query reporoot)/plz-out/plzexport build //src/core",
)
