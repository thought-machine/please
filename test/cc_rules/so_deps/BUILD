cc_shared_object(
    name = "lib",
    srcs = ["lib.cc"],
    hdrs = ["lib.h"],
)

cc_binary(
    name = "bin",
    srcs = ["main.cc"],
    deps = [":lib"],
)

gentest(
    name = "so_deps_test",
    test_cmd = "$DATA",
    no_test_output = True,
    data = [
        ":bin",
        ":lib",
    ],
)
