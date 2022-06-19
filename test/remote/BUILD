# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    binary = True,
    hashes = ["afabbbae6cb22045a7ee1d53edf0e298602ef6d41a3a2554e1e06555f9827eb4"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.6.9/elan",
)

remote_file(
    name = "mettle",
    binary = True,
    hashes = ["7f5b2914bd6bfa0461fc86ff14e086bf8fc42bff0f26ba766a7c633f8cd27ff2"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.6.9/mettle",
)

remote_file(
    name = "zeal",
    binary = True,
    hashes = ["c28f1cdc52820c78cfb137066cbe26c88ffc02a954690e37bc175d23eb44711a"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v9.6.9/zeal",
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
