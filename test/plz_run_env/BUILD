subinclude("//build_defs:plz_e2e_test")

sh_binary(
    name = "data_printer",
    main = "env_test.sh",
    data = ["//src:please", "wibble.txt"],
)

plz_e2e_test(
    name = "env_test",
    cmd = "plz run //test/plz_run_env:data_printer --env",
    expect_output_contains = "plz-out/bin/src/please test/plz_run_env/wibble.txt",
)
