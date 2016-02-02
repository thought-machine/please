# A little suite of targets that we use to test completions.

genrule(
    name = 'library',
    outs = ['lib.txt'],
    cmd = 'touch $OUT',
)

genrule(
    name = 'binary',
    outs = ['bin.sh'],
    cmd = 'touch $OUT',
    binary = True,
)

gentest(
    name = 'test',
    outs = ['test.sh'],
    cmd = 'touch $OUT',
    test_cmd = 'true',
    labels = ['manual'],
)
