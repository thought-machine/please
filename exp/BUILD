filegroup(
    name = "file",
    srcs = ["file.txt"],
    visibility = ["PUBLIC"],
)

build_rule(
    name = "module",
    srcs = ["module.txt"],
    cmd = "cp $SRC $OUT",
    outs = ["module_out.txt"],
    # relevant part
    provides = {
        'rq': ':file'
    }
)

# main rule
build_rule(
    name = "project",
    srcs = [":module"],
    cmd = "touch project; sleep 4",
    outs = ["project"],
    requires = ['rq'],
)
