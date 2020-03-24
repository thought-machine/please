# Test on filegroups using 'require' and 'provide'.
filegroup(
    name = "txt",
    srcs = ["a.txt"],
)

filegroup(
    name = "provider",
    provides = {
        "txt": ":txt",
    },
    deps = [":not_require_filegroup"],
)

filegroup(
    name = "require_filegroup",
    srcs = [":provider"],
    requires = ["txt"],
)

sh_test(
    name = "filegroup_require_test",
    src = "filegroup_require_test.sh",
    data = [":require_filegroup"],
)

filegroup(
    name = "not_require_filegroup",
    srcs = [":provider"],
)

sh_test(
    name = "filegroup_not_require_test",
    src = "filegroup_not_require_test.sh",
    data = [":not_require_filegroup"],
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
