subinclude("//test/build_defs")

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

plz_e2e_test(
    name = "pre_post_build_config_test",
    cmd = "plz build //test/parse/pre_post_build_config:target",
)

plz_e2e_test(
    name = "namedout_subinclude_test",
    cmd = "plz query alltargets //test/parse/namedout_subinclude:all",
)

plz_e2e_test(
    name = "label_list_subinclude_test",
    cmd = "plz query alltargets //test/parse/label_list_subinclude:all",
)
