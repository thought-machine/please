package(go_import_path = "github.com/thought-machine/please")

go_library(
    name = "importpath",
    srcs = ["importpath.go"],
)

go_test(
    name = "importpath_test",
    srcs = ["importpath_test.go"],
    external = True,
    deps = [
        ":importpath",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
