# This contains an arbitrary set of rules intended to exercise
# aspects of the cross-compilation code.

subinclude("///cc//build_defs:c")

c_library(
    name = "lib",
    srcs = ["lib.c"],
    hdrs = ["lib.h"],
    visibility = ["//test/cross_compile:all"],
)
