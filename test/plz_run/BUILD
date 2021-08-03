subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "env_test",
    expected_output = {
        "wibble.txt": "src/wibble.txt plz-out/bin/src/wibble.sh",
    },
    plz_command = "plz run //src:env_printer --env > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "env_test_in_tmp_dir",
    expected_output = {
        "wibble.txt": "src/wibble.txt src/wibble.sh",
    },
    plz_command = "plz run //src:env_printer --in_tmp_dir > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "std_in_test",
    expected_output = {
        "wibble.txt": "wibble",
    },
    plz_command = "echo //src:wibble | plz run - > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "std_in_parallel_test",
    expected_output = {
        "wibble.txt": "wibble\nwobble",
    },
    plz_command = "echo //src:wibble //src:wobble | plz run parallel - > wibble.txt",
    repo = "test_repo",
)

please_repo_e2e_test(
    name = "std_in_sequential_test",
    expected_output = {
        "wibble.txt": "wibble\nwobble",
    },
    plz_command = "echo //src:wibble //src:wobble | plz run sequential - > wibble.txt",
    repo = "test_repo",
)

# Test add_data functionality
#please_repo_e2e_test(
#    name = "_add_data_test",
#    expected_output = {
#        "_add_data_gen.txt": "//src:_add_data_gen //src:temp",
#    },
##    plz_command = "plz query deps //src:_add_data_gen > _add_data_gen.txt",
#    plz_command = "plz test //src:_add_data_gen",
#    repo = "test_repo",
#)

# Test add_data functionality
build_rule(
    name = "_add_data_test",
    test = True,
    no_test_output = True,
    test_cmd = "if [ $DATA != $PKG_DIR/temp.txt ]; then exit 1; else exit 0; fi;",
    pre_build = lambda name: add_data(name, "temp.txt"),
)

#genrule(
#    name = "temp",
#    cmd = "touch $OUT",
#    outs = ["foo.temp"],
#)
