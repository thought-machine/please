subinclude("//test/build_defs:build_defs")

filegroup(
    name = "export_e2e_test_build_def",
    srcs = ["please_export_e2e_test.build_defs"],
    visibility = ["//test/export/..."],
)

# Generic catch-all test on internal repo.
plz_e2e_test(
    name = "export_src_please_test",
    cmd = " && ".join([
        'CACHE_CONF="--override=cache.dir:$(plz query reporoot)/plz-out/test-cache"',
        'plz "$CACHE_CONF" export --output "$PLZ_EXPORT_DIR" //src/core:core',
        'plz "$CACHE_CONF" --repo_root="$PLZ_EXPORT_DIR" build //src/core:core',
    ]),
    pre_cmd = 'PLZ_EXPORT_DIR="$(mktemp -d)"',
)


# Generic catch-all test on internal repo - notrim.
plz_e2e_test(
    name = "export_src_please_notrim_test",
    cmd = " && ".join([
        'CACHE_CONF="--override=cache.dir:$(plz query reporoot)/plz-out/test-cache"',
        'plz "$CACHE_CONF" export --output "$PLZ_EXPORT_DIR" --notrim //src/core:core',
        'plz "$CACHE_CONF" --repo_root="$PLZ_EXPORT_DIR" build //src/core:core',
    ]),
    pre_cmd = 'PLZ_EXPORT_DIR="$(mktemp -d)"',
)
