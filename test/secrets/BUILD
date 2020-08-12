# Test on using secrets. Obviously it can't test any real secrets
# but it proves the system works.
gentest(
    name = "test_root_file",
    outs = ["test_root_file.sh"],
    cmd = 'echo "#!$SECRETS\nexec true" > "$OUT"',
    no_test_output = True,
    secrets = ["/bin/sh"],
    test_cmd = "$TEST",
)
