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

# Test to set string to tools argument
genrule(
    name = "tools_string",
    tools = "bash",
    cmd = "$TOOL -c \"echo 'String tools' > tools_string.txt\"",
    outs = [
        "tools_string.txt",
    ],
)

# Test to set string to test_tools argument
gentest(
    name = "test_tools_string",
    tools = {
        "SHELL": "bash",
    },
    no_test_output = True,
    test_tools = "bash",
    cmd = "$SHELL -c \"echo 'test string tools' > test_tools_strings.txt\"",
    outs = [
        "test_tools_strings.txt",
    ],
    test_cmd = "[ \"`cat $TEST`\" == \"test string tools\" ]"
)