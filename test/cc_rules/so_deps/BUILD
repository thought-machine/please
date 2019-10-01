cc_shared_object(
    name = "answer",
    srcs = ["lib.cc"],
    out = "libanswer.so",
    hdrs = ["lib.h"],
)

cc_binary(
    name = "bin",
    srcs = ["main.cc"],
    ldflags = ["-rpath='$ORIGIN'"],
    deps = [":answer"],
)

gentest(
    name = "so_deps_test",
    data = [
        ":answer",
        ":bin",
    ],
    no_test_output = True,
    test_cmd = "$DATA",
)
