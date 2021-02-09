subinclude("//build_defs:plz_e2e_test")

sh_binary(
    name = "env_printer",
    data = [
        "wibble.txt",
        "//src:please",
    ],
    main = "env_printer.sh",
)

sh_cmd(
    name = "wibble",
    cmd = "echo wibble",
)

sh_cmd(
    name = "wobble",
    cmd = "sleep 0.1 && echo wobble",
)

plz_e2e_test(
    name = "env_test",
    cmd = "plz run //test/plz_run:env_printer --env",
    expect_output_contains = "test/plz_run/wibble.txt plz-out/bin/src/please",
)

plz_e2e_test(
    name = "std_in_test",
    cmd = "echo //test/plz_run:wibble | plz run -",
    expect_output_contains = "wibble",
)

plz_e2e_test(
    name = "std_in_parallel_test",
    cmd = "echo //test/plz_run:wibble //test/plz_run:wobble | plz run parallel -",
    expect_output_contains = "wibble\nwobble",
)

plz_e2e_test(
    name = "std_in_sequential_test",
    cmd = "echo //test/plz_run:wibble //test/plz_run:wobble | plz run sequential -",
    expect_output_contains = "wibble\nwobble",
)