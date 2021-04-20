# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    binary = True,
    hashes = ["b1f8b97a837b5c88dbb4f6841603fa51748a22277aeada6bae10f43a3e063b1c"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.0.1/elan",
)

remote_file(
    name = "mettle",
    binary = True,
    hashes = ["dee3b9e76e083adaa93a44e03c53c04d9e06e6fe9f581b717326c04e75d9ce59"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.0.1/mettle",
)

remote_file(
    name = "zeal",
    binary = True,
    hashes = ["a1f43fadc76403afbcc7a04710904812a49a7cd9a7245d92a8c2c870fcde6649"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.0.1/zeal",
)

sh_cmd(
    name = "run_elan",
    srcs = [":elan"],
    cmd = "mkdir -p plz-out/elan && exec $(out_location :elan) -s file://\\\\$PWD/plz-out/elan -v warning --log_file plz-out/log/elan.log --port 7777 --admin_disabled",
)

sh_cmd(
    name = "run_mettle",
    srcs = [":mettle"],
    cmd = "exec $(out_location :mettle) dual -s 127.0.0.1:7777 -d plz-out/mettle -v warning --log_file plz-out/log/mettle.log --browser http://127.0.0.1:7779 --num_workers 8 --port 7778 --admin_disabled",
)

sh_cmd(
    name = "run_zeal",
    srcs = [":zeal"],
    cmd = "exec $(out_location :zeal) -v warning -s 127.0.0.1:7777 --log_file plz-out/log/zeal.log --port 7776 --admin_disabled",
)
