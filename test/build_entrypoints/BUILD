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
    build_entry_point = ["bash", "--noprofile", "--norc", "-u", "-o", "pipefail"],
    build_entry_point_exit_on_error_args = ["-e"],
    build_entry_point_exec_command_args = ["-c"],
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
    build_entry_point = ["$(exe :main)"],
    tools = [":main"],
)

sh_test(
    name = "binary_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":binary_entrypoint"],
)

linux_libc="gnu"
if CONFIG.OS == "linux":
    genrule(
        name = "libc_impl",
        outs = ["libc_impl"],
        cmd = """
    lddVersionOut="$(ldd --version 2>&1 || true)"
    if echo "$lddVersionOut" | grep -i "GNU\|GLIBC"; then
        echo "libc='gnu'" > $OUTS
        exit
    fi

    if echo "$lddVersionOut" | grep -i "musl"; then
        echo "libc='musl'" > $OUTS
        exit
    fi

    echo "$lddVersionOut"
    exit 1
        """,
    )

    subinclude(":libc_impl")
    linux_libc=libc


nushell_version = "0.104.1"
nushell_release_asset_suffix_by_os_arch = {
    "linux_amd64": f"-x86_64-unknown-linux-{linux_libc}.tar.gz",
    "linux_arm64": f"-aarch64-unknown-linux-{linux_libc}.tar.gz",
    "darwin_amd64": "-x86_64-apple-darwin.tar.gz",
    "darwin_arm64": "-aarch64-apple-darwin.tar.gz",
}
nushell_release_asset_suffix = nushell_release_asset_suffix_by_os_arch[CONFIG.OS+"_"+CONFIG.ARCH]
nushell_release_asset_extracted_suffix = nushell_release_asset_suffix.split(".")[0]

remote_file(
    name = "nu",
    url = f"https://github.com/nushell/nushell/releases/download/{nushell_version}/nu-{nushell_version}{nushell_release_asset_suffix}",
    extract = True,
    binary = True,
    exported_files = [f"nu-{nushell_version}{nushell_release_asset_extracted_suffix}/nu"],
)

build_rule(
    name = "nushell_entrypoint",
    outs = ["nushell_entrypoint.txt"],
    build_entry_point = ["$(exe :nu)", "--no-config-file", "--no-history"],
    build_entry_point_interactive_args = ["--execute", "$env.config.show_banner = false"],
    build_entry_point_exec_command_args = ["--commands"],
    cmd = "$env.NAME | save $env.OUTS",
    tools = [":nu"],
)

sh_test(
    name = "nushell_entrypoint_test",
    src = "test_build_entrypoint.sh",
    data = [":nushell_entrypoint"],
)
