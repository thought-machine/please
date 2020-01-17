sh_library(
    name = "sh_lib",
    src = "sh_lib.sh",
)

sh_binary(
    name = "sh_binary",
    main = "sh_binary.sh",
    deps = [
        ":sh_lib",
    ],
)

sh_test(
    name = "sh_test",
    src = ":sh_binary",
)

sh_cmd(
    name = "sh_cmd",
    cmd = [
        "TEST_VAR=\"$@\"",
        "[ \"$TEST_VAR\" = \"test_value\" ] || (echo \"Expected a value expansion not to happen\" >&2 && exit 11)",
    ],
)

sh_test(
    name = "sh_cmd_test",
    src = ":sh_cmd",
    flags = "test_value",
)
