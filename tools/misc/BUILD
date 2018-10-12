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

python_binary(
    name = "gen_release",
    main = "gen_release.py",
    deps = [
        "//third_party/python:absl",
        "//third_party/python:colorlog",
        "//third_party/python:requests",
    ],
)
