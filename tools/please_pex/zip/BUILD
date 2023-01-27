go_library(
    name = "zip",
    srcs = ["writer.go"],
    visibility = [
        "//tools/jarcat:all",
        "//tools/please_pex/pex",
    ],
    deps = [
        "///third_party/go/gopkg.in_op_go-logging.v1//:go-logging.v1",
        "//src/fs",
    ],
)

go_test(
    name = "writer_test",
    srcs = ["writer_test.go"],
    data = [
        "test_data",
        "test_data_2",
        "test_data_3",
    ],
    deps = [
        ":zip",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
