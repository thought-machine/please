subinclude("//test/build_defs")

plz_e2e_test(
    name = "multiple_sources",
    cmd = "plz query whatinputs test/whatinputs/test_package/foo.txt test/whatinputs/test_package/foo.txt",
    expected_failure = False,
    expected_output = "multiple_sources.txt",
)

plz_e2e_test(
    name = "source_and_no_source",
    cmd = "plz query whatinputs making/up/a/file/path/should/be/easy test/whatinputs/test_package/foo.txt",
    expected_failure = True,
    expect_output_contains = "making/up/a/file/path/should/be/easy is not a source to any current target",
)

plz_e2e_test(
    name = "source_and_no_source_print",
    cmd = "plz query whatinputs making/up/a/file/path/should/be/easy test/whatinputs/test_package/foo.txt --echo_files",
    expected_failure = True,
    expect_output_contains = "making/up/a/file/path/should/be/easy is not a source to any current target",
)
