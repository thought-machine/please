subinclude("///shell//build_defs:shell")

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

filegroup(
    name = "pgo",
    srcs = ["default.pgo"],
    visibility = ["//src/..."],
)

filegroup(
    name = "go_mod",
    srcs = ["go.mod"],
    visibility = ["//third_party/go/..."],
)
