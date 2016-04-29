# TODO(pebers): Revert to the github version once plz is public.
# subinclude('https://github.com/thought-machine/please/raw/master/build_defs/plz_e2e_test.build_defs')
subinclude('https://s3-eu-west-1.amazonaws.com/please-build/plz_e2e_test.build_defs')

plz_e2e_test(
    name = 'remote_subinclude_test',
    cmd = 'plz query alltargets //test/remote_rules:all',
    expect_output_contains = 'remote_subinclude_test',
)
