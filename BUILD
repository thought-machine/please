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

filegroup(
    name = "pleasew",
    srcs = ["pleasew"],
    binary = True,
    visibility = ["//src/assets/..."],
)

sh_cmd(
    name = "autofix",
    cmd = "plz fmt -w && gofmt -s -w src tools test && plz run parallel --include codegen",
)

github_repo(
    name = "pleasings",
    repo = "thought-machine/pleasings",
    revision = "v1.1.0",
)
