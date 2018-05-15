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
