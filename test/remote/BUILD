# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    binary = True,
    hashes = ["dd8f7d546d270a198b271841c0d687cd366c770589d34e2f654964d6447fdbd8"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.5.1/elan",
)

remote_file(
    name = "mettle",
    binary = True,
    hashes = ["1a28d2de46a626353f0ba5ab350a6a9d2f52303e19d016f2e36273054a62575e"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.5.1/mettle",
)

remote_file(
    name = "zeal",
    binary = True,
    hashes = ["2831f71f4eaedf6452d05e5fd1cf00d45f007d55112a3f74b649ef8a341f87e1"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.5.1/zeal",
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
