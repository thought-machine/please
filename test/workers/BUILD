go_binary(
    name = "test_bin",
    srcs = ["test.go"],
    deps = ["//third_party/go:logging"],
)

go_binary(
    name = "worker",
    srcs = ["worker.go"],
    visibility = [
        "//test/python_rules/...",
    ],
    deps = [
        "//third_party/go:logging",
        "//third_party/go:protobuf",
    ],
)

gentest(
    name = "worker_test",
    data = [":test_bin"],
    no_test_output = True,
    sandbox = False,
    test_cmd = "$(worker :worker) && $DATA; [ ! $? -eq 0 ]",
    tools = [":worker"],
)
