subinclude("///cc//build_defs:c", "///cc//build_defs:cc")

c_library(
    name = "sandbox",
    srcs = ["sandbox.c"],
    hdrs = ["sandbox.h"],
)

c_binary(
    name = "please_sandbox",
    srcs = ["main.c"],
    visibility = ["PUBLIC"],
    deps = [":sandbox"],
)

c_binary(
    name = "nonet_sandbox",
    srcs = ["nonet_main.c"],
    deps = [":sandbox"],
)

c_binary(
    name = "noproc_sandbox",
    srcs = ["noproc_main.c"],
    deps = [":sandbox"],
)

c_binary(
    name = "nonetproc_sandbox",
    srcs = ["nonetproc_main.c"],
    deps = [":sandbox"],
)

go_test(
    name = "sandbox_test",
    srcs = ["sandbox_test.go"],
    data = {
        "tm_sandbox": ":tm_sandbox",
        "nonet_sandbox": ":nonet_sandbox",
        "noproc_sandbox": ":noproc_sandbox",
        "nonetproc_sandbox": ":nonetproc_sandbox",
    },
    labels = [
        "localonly",
        "manual",
    ],
    sandbox = False,
    deps = [
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)

cc_test(
    name = "sandbox_test",
    srcs = ["sandbox_test.cc"],
    deps = [
        ":sandbox",
    ],
)
