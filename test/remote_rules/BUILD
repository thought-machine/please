subinclude('https://raw.githubusercontent.com/thought-machine/please/master/build_defs/plz_e2e_test.build_defs')

plz_e2e_test(
    name = 'remote_subinclude_test',
    cmd = 'plz query alltargets //test/remote_rules:all',
    expect_output_contains = 'remote_subinclude_test',
)
