subinclude("///cc//build_defs:cc")

cc_library(
    name = "lib1",
    hdrs = ["lib1/include/lib.hpp"],
    includes = ["lib1/include"],
)

cc_library(
    name = "another_lib",
    srcs = ["another_lib.cpp"],
    deps = [":lib1"],
)

cc_test(
    name = "lib_test",
    srcs = ["lib_test.cpp"],
    deps = [":another_lib"],
)
