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
    srcs = [".plzconfig_freebsd_amd64", ".plzconfig.e2e"],
)
