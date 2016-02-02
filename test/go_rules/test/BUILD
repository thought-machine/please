# For testing the go build commands, where we need to be careful about things
# being in subdirectories.
go_library(
    name = 'test',
    srcs = ['test.go'],
    visibility = ['//test/...'],
    test_only = True,
)
