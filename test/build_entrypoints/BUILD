build_rule(
    name = "default_entrypoint",
    outs = ["default_entrypoint.txt"],
    # Default is Bash
    cmd = "printf '%s' \"$NAME\" > $OUTS",
)

sh_test(
    name = "default_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":default_entrypoint"],
)

build_rule(
    name = "bash_entrypoint",
    outs = ["bash_entrypoint.txt"],
    build_entrypoint = {
        "entrypoint": ["bash", "--noprofile", "--norc", "-u", "-o", "pipefail"],
        "exit_on_error_flags": ["-e"],
        "exec_command_flags": ["-c"],
    },
    cmd = "printf '%s' \"$NAME\" > $OUTS",
)

sh_test(
    name = "bash_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":bash_entrypoint"],
)

subinclude("///go//build_defs:go")

go_binary(
    name = "main",
    srcs = ["main.go"],
)

build_rule(
    name = "binary_entrypoint",
    outs = ["binary_entrypoint.txt"],
    build_entrypoint = {
        "entrypoint": ["$(exe :main)"],
    },
    tools = [":main"],
)

sh_test(
    name = "binary_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":binary_entrypoint"],
)

remote_file(
    name = "nu",
    url = "https://github.com/nushell/nushell/releases/download/0.104.1/nu-0.104.1-x86_64-unknown-linux-gnu.tar.gz",
    extract = True,
    binary = True,
    exported_files = ["nu-0.104.1-x86_64-unknown-linux-gnu/nu"],
)

build_rule(
    name = "nushell_entrypoint",
    outs = ["nushell_entrypoint.txt"],
    build_entrypoint = {
        "entrypoint": ["$(exe :nu)", "--no-config-file", "--no-history"],
        "interactive_flags": ["--execute", "$env.config.show_banner = false"],
        "exec_command_flags": ["--commands"],
    },
    cmd = "$env.NAME | save $env.OUTS",
    tools = [":nu"],
)

sh_test(
    name = "nushell_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":nushell_entrypoint"],
)

remote_file(
    name = "powershell_toolchain",
    url = "https://github.com/PowerShell/PowerShell/releases/download/v7.5.2/powershell-7.5.2-linux-x64.tar.gz",
    extract = True,
    out = "powershell_toolchain",
    entry_points = {
        "pwsh": "powershell_toolchain/pwsh",
    },
    binary = True,
)

build_rule(
    name = "powershell_entrypoint",
    outs = ["powershell_entrypoint.txt"],
    build_entrypoint = {
        "entrypoint": ["$(exe :powershell_toolchain)/pwsh", "-NoProfile"],
        "interactive_flags": ["-Interactive"],
        "exec_command_flags": ["-Command"],
    },
    cmd = '$Env:NAME | Out-File -FilePath $Env:OUTS',
    tools = [":powershell_toolchain"],
)

sh_test(
    name = "powershell_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":powershell_entrypoint"],
)
