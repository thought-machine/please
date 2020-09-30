package(default_visibility = ["//test/..."])

sh_binary(
    name = "content_checker",
    main = "test_file_content.sh",
)

filegroup(
    name = "build_defs",
    srcs = glob(["*.build_defs"]),
)

filegroup(
    name = "freebsd_config",
    srcs = [".plzconfig_freebsd_amd64"],
)
