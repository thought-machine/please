go_library(
    name = "fs",
    srcs = ["fs.go"],
    deps = [
        "///third_party/go/github.com_bazelbuild_remote-apis-sdks//go/pkg/client",
        "///third_party/go/github.com_bazelbuild_remote-apis-sdks//go/pkg/digest",
        "///third_party/go/github.com_bazelbuild_remote-apis//build/bazel/remote/execution/v2",
        "///third_party/go/google.golang.org_protobuf//proto",
    ],
)

go_test(
    name = "fs_test",
    srcs = ["fs_test.go"],
    deps = [
        ":fs",
        "///third_party/go/github.com_bazelbuild_remote-apis-sdks//go/pkg/client",
        "///third_party/go/github.com_bazelbuild_remote-apis-sdks//go/pkg/digest",
        "///third_party/go/github.com_bazelbuild_remote-apis//build/bazel/remote/execution/v2",
        "///third_party/go/github.com_golang_protobuf//proto",
        "///third_party/go/github.com_golang_protobuf//ptypes/wrappers",
        "///third_party/go/github.com_stretchr_testify//assert",
        "///third_party/go/github.com_stretchr_testify//require",
    ],
)
