cc_shared_object(
    name = "answer",
    out = "libanswer.so",
    srcs = ["lib.cc"],
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
    test_cmd = "$DATA",
    no_test_output = True,
    data = [
        ":bin",
        ":answer",
    ],
)
