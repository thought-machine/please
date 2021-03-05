c_library(
    name = "sandbox",
    srcs = ["sandbox.c"],
    hdrs = ["sandbox.h"],
)

c_binary(
    name = "please_sandbox",
    srcs = ["main.c"],
    deps = [":sandbox"],
    static = CONFIG.get('STATIC_SANDBOX') is not None,
    visibility = ["PUBLIC"],
)

c_binary(
    name = "nonet_sandbox",
    srcs = ["nonet_main.c"],
    deps = [":sandbox"],
    static = CONFIG.get('STATIC_SANDBOX') is not None,
    visibility = ["PUBLIC"],
)
