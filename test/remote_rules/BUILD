subinclude('https://raw.githubusercontent.com/thought-machine/please/42d89f7a94350d01d9de6e4a3a476a18e14b09f2/build_defs/plz_e2e_test.build_defs')

plz_e2e_test(
    name = 'remote_subinclude_test',
    cmd = 'plz query alltargets //test/remote_rules:all',
    expect_output_contains = 'remote_subinclude_test',
    labels = ['manual'],  # Disabled until we update the build defs in the repo.
    sandbox = False,
)
