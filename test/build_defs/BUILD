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
    name = "freebsd_config",
    srcs = [".plzconfig_freebsd_amd64"],
)

filegroup(
    name = "e2e_config",
    srcs = [".plzconfig.e2e"],
)