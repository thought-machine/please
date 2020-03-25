# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    url = "https://github.com/thought-machine/please-servers/releases/download/v2.3.0/elan",
    hashes = ["ac113612e582055302fc732512a31aeb548f6c8837da39f03067419862fb7c14"],
    binary = True,
)

remote_file(
    name = "mettle",
    url = "https://github.com/thought-machine/please-servers/releases/download/v2.3.0/mettle",
    hashes = ["0f65f009a3dd71e120ec9cba233f3a756a7a17b21b53e8297fd49ea1c38886ad"],
    binary = True,
)

remote_file(
    name = "zeal",
    url = "https://github.com/thought-machine/please-servers/releases/download/v2.3.0/zeal",
    hashes = ["b61f5f23b1c82e9aac4f5b9ad282e03112eeff8fe1441c7bf2e960fefd3c6c1b"],
    binary = True,
)

sh_cmd(
    name = "run_elan",
    cmd = "mkdir -p plz-out/elan && exec $(out_location :elan) -s file://\\\\$PWD/plz-out/elan --log_file plz-out/log/elan.log",
    srcs = [":elan"],
)

sh_cmd(
    name = "run_mettle",
    cmd = "exec $(out_location :mettle) dual -s 127.0.0.1:7777 -q mem://requests -r mem://responses -m 13434 -d plz-out/mettle --log_file plz-out/log/mettle.log --browser http://127.0.0.1:7779 --sandbox ~/.please/please_sandbox --num_workers 8",
    srcs = [":mettle"],
)

sh_cmd(
    name = "run_zeal",
    cmd = "exec $(out_location :zeal) -s 127.0.0.1:7777 --log_file plz-out/log/zeal.log",
    srcs = [":zeal"],
)
