# Reduces the published codelabs to a plan that test/windows/run_codelabs.ps1 replays on a real
# Windows machine. See script/parse.go for why it refuses to guess.
go_binary(
    name = "codelab_script",
    srcs = ["main.go"],
    visibility = ["//test/windows/..."],
    deps = [
        "///third_party/go/github.com_peterebden_go-cli-init_v5//flags",
        "//test/windows/codelab_script/script",
    ],
)
