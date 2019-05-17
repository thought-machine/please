subinclude("//build_defs:plz_e2e_test")

plz_e2e_test(
    name = "subinclude_visibility_test",
    cmd = "plz query alltargets //test/parse/subinclude_visibility:all",
    expect_output_contains = "isn't visible",
    expected_failure = True,
)

plz_e2e_test(
    name = "inline_subinclude_test",
    cmd = "plz query alltargets //test/parse/inline_subinclude:all",
)
