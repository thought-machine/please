go_library(
    name = "assets",
    srcs = [":bindata"],
    visibility = ["//src/..."],
)

go_bindata(
    name = "bindata",
    srcs = [":bindata_files"],
)

# If you update this, you must also update bindata.go,
filegroup(
    name = "bindata_files",
    srcs = [
        "//:pleasew",
        "//tools/misc:completion_script",
    ],
)

sh_cmd(
    name = "generate",
    cmd = "cp $(out_location :bindata) src/assets/bindata.go",
    data = [":bindata"],
)

sh_test(
    name = "up_to_date_test",
    src = "up_to_date_test.sh",
    data = {
        "BIN_DATA": [":bindata"],
        "GENERATED": ["bindata.go"],
    },
)
