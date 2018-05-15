# Test on filegroups using 'require'.
filegroup(
    name = "require_filegroup",
    srcs = ["//test/proto_rules:test_proto"],
    requires = ["cc_hdrs"],
)

sh_test(
    name = "filegroup_require_test",
    src = "filegroup_require_test.sh",
    data = [":require_filegroup"],
)

# Test on having a symlink to an output
genrule(
    name = "test_symlinks",
    outs = [
        "test1.txt",
        "test2.txt",
    ],
    cmd = "echo \"test\" > test1.txt && ln -s test1.txt test2.txt",
)
