# A little suite of targets that we use to test completions.

genrule(
    name = "library",
    outs = ["lib.txt"],
    cmd = 'touch "$OUT"',
)

genrule(
    name = "binary",
    outs = ["bin.sh"],
    binary = True,
    cmd = 'touch "$OUT"',
)

gentest(
    name = "test",
    outs = ["test.sh"],
    cmd = 'touch "$OUT"',
    labels = ["manual"],
    test_cmd = "true",
)
