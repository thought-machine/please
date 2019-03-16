filegroup(
    name = "version",
    srcs = ["VERSION"],
    visibility = ["PUBLIC"],
)

filegroup(
    name = "changelog",
    srcs = ["ChangeLog"],
    visibility = ["PUBLIC"],
)

genrule(
    name = "bootstrap",
    srcs = ["bootstrap.sh"],
    outs = ["bootstrap.sh"],
    cmd = "sed 's/EXCLUDES=\"\"/EXCLUDES=\"%s\"/' $SRC > $OUT" % CONFIG.get("EXCLUDETEST", ""),
    binary = True,
)

filegroup(
    name = "install",
    srcs = ["install.sh"],
    binary = True,
    deps = ["//package:installed_files"],
)
