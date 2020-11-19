subinclude("//build_defs:plz_e2e_test")

sh_binary(
    name = "env_printer",
    data = [
        "wibble.txt",
        "//src:please",
    ],
    main = "env_printer.sh",
)

plz_e2e_test(
    name = "env_test",
    cmd = "plz run //test/plz_run_env:env_printer --env",
    expect_output_contains = "plz-out/bin/src/please test/plz_run_env/wibble.txt",
)
