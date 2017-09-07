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

filegroup(
    name = 'engines',
    srcs = ['engines.build_defs'],
    visibility = ['PUBLIC'],
)

filegroup(
    name = 'fpm',
    srcs = ['fpm.build_defs'],
    visibility = ['//package:all'],
)

filegroup(
    name = 'multiversion_wheel',
    srcs = ['multiversion_wheel.build_defs'],
    visibility = ['//third_party/python:all'],
)

genrule(
    name = 'version',
    srcs = ['//:version'],
    outs = ['version.build_defs'],
    cmd = 'echo "VERSION = \'`cat $SRCS`\'" > $OUT',
    visibility = ['PUBLIC'],
)
