# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    binary = True,
    hashes = ["6e97cb8530d3202e599486538a31d9ef63e0e19e5720e820d6cae4067e54c9aa"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v7.3.5/elan",
)

remote_file(
    name = "mettle",
    binary = True,
    hashes = ["36a71e9a15019dc4ab765a81f7ec16de0ad531ea7c41ed3e6491c40f50c56a72"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v7.3.5/mettle",
)

remote_file(
    name = "zeal",
    binary = True,
    hashes = ["cecf4a386d1787613af5b910f354af120ac3f8e5ed49ca7c669166fe3b6fa4f4"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v7.3.5/zeal",
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
