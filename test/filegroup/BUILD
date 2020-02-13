subinclude("//build_defs:plz_e2e_test")

genrule(
    name = "gen",
    outs = ["gen.txt"],
    # This being nondeterministic would normally be a bad idea, but here makes the test work better.
    cmd = 'dd if=/dev/urandom of="$OUT" bs=10000 count=1',
    visibility = ["//test/filegroup/many:all"],
)

plz_e2e_test(
    name = "filegroup_concurrent_build_test",
    cmd = "plz clean //test/filegroup:gen && plz build //test/filegroup/many:all",
)
