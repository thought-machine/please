package(default_visibility = ["//test/..."])

sh_binary(
    name = "content_checker",
    main = "test_file_content.sh",
    visibility = ["//test/..."],
)

filegroup(
    name = "build_defs",
    srcs = glob(["*.build_defs"]),
)

filegroup(
    name = "base_config",
    srcs = [
        ".plzconfig.e2e",
        ".plzconfig_freebsd_amd64",
    ],
)
