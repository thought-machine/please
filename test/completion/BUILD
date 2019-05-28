subinclude("//build_defs:plz_e2e_test")

cc_library(
    name = "lib",
    srcs = ["lib.cc"],
)

cc_binary(
    name = "bin",
    srcs = ["bin.cc"],
)

# plz test should be able to complete tests in this directory
plz_e2e_test(
    name = "plz_test_completion_test",
    cmd = "plz test //test/completion:plz_test_",
    completion = True,
    expect_output_contains = "//test/completion:plz_test_completion_test",
)

# plz run should not complete tests.
plz_e2e_test(
    name = "plz_run_anti_test",
    cmd = "plz run //test/completion:plz_test_",
    completion = True,
    expect_output_doesnt_contain = "//test/completion:plz_test_completion_test",
)

# plz build should complete libraries
plz_e2e_test(
    name = "plz_build_lib_test",
    cmd = "plz build //test/completion:",
    completion = True,
    expect_output_contains = "//test/completion:lib",
)

# and binaries
plz_e2e_test(
    name = "plz_build_bin_test",
    cmd = "plz build //test/completion:",
    completion = True,
    expect_output_contains = "//test/completion:bin",
)

# plz run should not complete libraries
plz_e2e_test(
    name = "plz_run_lib_anti_test",
    cmd = "plz run //test/completion:",
    completion = True,
    expect_output_doesnt_contain = "//test/completion:lib",
)

# and binaries
plz_e2e_test(
    name = "plz_run_bin_test",
    cmd = "plz run //test/completion:",
    completion = True,
    expect_output_contains = "//test/completion:bin",
)

# plz run should complete files after --
plz_e2e_test(
    name = "plz_run_args_test",
    cmd = "plz run //test/completion:bin -- test/completion/",
    completion = True,
    expect_output_contains = "test/completion/bin.cc",
)
