subinclude("//test/build_defs")

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

hash_filegroup(
    name = "hashed",
    srcs = ["hash_filegroup_test.txt"],
)

sh_test(
    name = "hash_filegroup_test",
    src = "hash_filegroup_test.sh",
    data = [":hashed"],
)

text_file(
    name = "wibble",
    content = "wibblewibblewibble",
)

text_file(
    name = "wobble",
    content = "wobblewobblewobble",
)


filegroup(
    name = "named",
    srcs = {
        "wibble": [":wibble"],
        "wobble": [":wobble"],
    }
)

sh_test(
    name = "named_test",
    src = "named_test.sh",
    data = {
        "wibble": [":named|wibble"],
        "wobble": [":named|wobble"],
        "all": [":named"],
    },
)
