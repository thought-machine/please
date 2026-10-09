filegroup(
    name = "completion_script",
    srcs = ["plz_complete.sh"],
    visibility = ["PUBLIC"],
)

python_binary(
    name = "gen_release",
    labels = ["hlink:plz-out/pkg"],
    main = "gen_release.py",
    deps = [
        "//:changelog",
        "//:version",
        "//third_party/python:absl",
        "//third_party/python:colorlog",
        "//third_party/python:requests",
    ],
)

python_test(
    name = "gen_release_test",
    srcs = ["gen_release_test.py"],
    data = ["data"],
    deps = [":gen_release"],
)

sh_binary(
    name = "lint",
    main = "lint.sh",
)

python_binary(
    name = "plz_out_usage",
    main = "plz_out_usage.py",
)

python_test(
    name = "plz_out_usage_test",
    srcs = ["plz_out_usage_test.py"],
    deps = [":plz_out_usage"],
)
