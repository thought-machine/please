subinclude("///cc//build_defs:c", "///cc//build_defs:cc")

c_library(
    name = "sandbox",
    srcs = ["sandbox.c"],
    hdrs = ["sandbox.h"],
)

c_binary(
    name = "please_sandbox",
    srcs = ["main.c"],
    static = (CONFIG.get("STATIC_SANDBOX") is not None),
    visibility = ["PUBLIC"],
    deps = [":sandbox"],
)

c_binary(
    name = "nonet_sandbox",
    srcs = ["nonet_main.c"],
    static = (CONFIG.get("STATIC_SANDBOX") is not None),
    visibility = ["PUBLIC"],
    deps = [":sandbox"],
)

cc_test(
    name = "sandbox_test",
    srcs = ["sandbox_test.cc"],
    deps = [
        ":sandbox",
    ],
)
