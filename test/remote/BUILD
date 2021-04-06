# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    binary = True,
    hashes = ["bf54d35dad4a4def772ce04a668164e028d79cb19ff51d176d9c8322de1e2028"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v8.8.4/elan",
)

remote_file(
    name = "mettle",
    binary = True,
    hashes = ["7dad94c1034a25bb49baf503bf518e7237980465bc4dd355f069bed01fdc0444"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v8.8.4/mettle",
)

remote_file(
    name = "zeal",
    binary = True,
    hashes = ["a293cc2e77f72d1379d2dcdb12e52844b3339fad107c875bacba4791badc4230"],
    url = "https://github.com/thought-machine/please-servers/releases/download/v8.8.4/zeal",
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
