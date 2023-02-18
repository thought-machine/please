# For testing the go build commands, where we need to be careful about things
# being in subdirectories.
go_library(
    name = "test",
    srcs = ["test.go"],
    test_only = True,
    visibility = ["//test/..."],
)

# Tests a go 'external test', where you can have files in the same directory
# with a _test suffix.
go_test(
    name = "external_test",
    srcs = ["external_test.go"],
    definitions = {
        "github.com/thought-machine/please/test/go_rules/test.Var": "var",
        "github.com/thought-machine/please/test/go_rules/test.Var2": "var1 var2",
        "github.com/thought-machine/please/test/go_rules/test.ExecGitShow": git_show("%ct"),
        "github.com/thought-machine/please/test/go_rules/test.ExecGitState": git_state(
            clean_label = "shiny",
            dirty_label = "not shiny",
        ),
        "github.com/thought-machine/please/test/go_rules/test.ExecGitCommit": git_commit(),
        "github.com/thought-machine/please/test/go_rules/test.ExecGitBranchFull": git_branch(short = False),
        "github.com/thought-machine/please/test/go_rules/test.ExecGitBranchShort": git_branch(),
    },
    external = True,
    labels = ["no_cirrus"],  # Git branches aren't happening as expected here.
    deps = [
        ":test",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
