subinclude("///cc//build_defs:cc")

# Tests that a sequence of dependencies works as expected.
cc_library(
    name = "lib1",
    srcs = ["lib1.cc"],
    hdrs = ["lib1.h"],
)

cc_library(
    name = "lib2",
    srcs = [
        "lib2_1.cc",
        "lib2_2.cc",
    ],
    hdrs = ["lib2.h"],
    deps = [":lib1"],
)

cc_library(
    name = "lib3",
    srcs = ["lib3.cc"],
    hdrs = ["lib3.h"],
    deps = [":lib2"],
)

cc_test(
    name = "deps_test",
    srcs = ["deps_test.cc"],
    deps = [":lib3"],
)
