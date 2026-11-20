# Various helpers for testing remote execution.
remote_file(
    name = "elan",
    binary = True,
    hashes = ["89de87d8c592de4248142e4b948d4a5e2291a4dd0504a181a2bc9c31eb5821ab"],
    url = "https://github.com/please-build/mettle/releases/download/v11.19.4/elan",
)

remote_file(
    name = "mettle",
    binary = True,
    hashes = ["c199cd50ca14ce544b92643fb45d75d3468f21bfbf15b2b9fd3e87b8b585d958"],
    url = "https://github.com/please-build/mettle/releases/download/v11.19.4/mettle",
)

remote_file(
    name = "zeal",
    binary = True,
    hashes = ["5d81482f671c14675c8b45f3119df0e53617eada30047c16f23c2962e2d077a9"],
    url = "https://github.com/please-build/mettle/releases/download/v11.19.4/zeal",
)

sh_cmd(
    name = "run_elan",
    srcs = [":elan"],
    cmd = "mkdir -p plz-out/elan && exec $(out_location :elan) -s 'file://'\\\\$PWD'/plz-out/elan?metadata=skip&no_tmp_dir=true' -v error --log_file plz-out/log/elan.log --port 7777 --admin_disabled",
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
