filegroup(
    name = "wrapper_script",
    srcs = ["pleasew"],
    visibility = ["PUBLIC"],
)

filegroup(
    name = "completion_script",
    srcs = ["plz_complete.sh"],
    visibility = ["PUBLIC"],
)

genrule(
    name = "version",
    srcs = ["//build_defs:version"],
    outs = ["version.py"],
    cmd = "mv $SRCS $OUT",
)

python_binary(
    name = "gen_release",
    labels = ["hlink:plz-out/pkg"],
    main = "gen_release.py",
    deps = [
        ":version",
        "//third_party/python:absl",
        "//third_party/python:colorlog",
        "//third_party/python:requests",
    ],
)
