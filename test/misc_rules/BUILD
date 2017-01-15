# Test on filegroups using 'require'.
filegroup(
    name = 'require_filegroup',
    srcs = ['//test/proto_rules:test_proto'],
    requires = ['cc_hdrs'],
)

sh_test(
    name = 'filegroup_require_test',
    src = 'filegroup_require_test.sh',
    data = [':require_filegroup'],
)
