# For testing the go build commands, where we need to be careful about things
# being in subdirectories.
go_library(
    name = 'test',
    srcs = ['test.go'],
    test_only = True,
    visibility = ['//test/...'],
)

go_library(
    name = 'generate_test',
    srcs = ['generate.go'],
    go_tools = [
        '//third_party/go:stringer',
    ],
    visibility = ['//test/...'],
)
