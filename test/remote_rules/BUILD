subinclude('https://raw.githubusercontent.com/thought-machine/please/01ba4a0bb5740d61469b12df23d9c793218a7f50/build_defs/plz_e2e_test.build_defs')

plz_e2e_test(
    name = 'remote_subinclude_test',
    cmd = 'plz query alltargets //test/remote_rules:all',
    expect_output_contains = 'remote_subinclude_test',
    sandbox = False,
)
