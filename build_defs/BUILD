filegroup(
    name = 'go_bindata',
    srcs = ['go_bindata.build_defs'],
    visibility = ['PUBLIC'],
)

filegroup(
    name = 'plz_e2e_test',
    srcs = ['plz_e2e_test.build_defs'],
    visibility = ['//test/...'],
)

genrule(
    name = 'version',
    srcs = ['//:version'],
    outs = ['version.build_defs'],
    cmd = 'echo "VERSION = \'`cat $SRCS`\'" > $OUT',
    visibility = ['PUBLIC'],
)
