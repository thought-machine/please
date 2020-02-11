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
    binary = True,
    cmd = "sed 's/EXCLUDES=\"\"/EXCLUDES=\"%s\"/' $SRC > \"$OUT\"" % CONFIG.get("EXCLUDETEST", ""),
)

filegroup(
    name = "install",
    srcs = ["install.sh"],
    binary = True,
    deps = ["//package:installed_files"],
)

# This is used as part of bootstrap, and is used from here to avoid subtle issues with remote execution.
filegroup(
    name = "jarcat_unzip",
    srcs = ["//tools/jarcat:jarcat_unzip"],
    binary = True,
    visibility = ["//third_party/go:all"],
)
