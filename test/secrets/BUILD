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

tildes_replaced_test = r'''
echo "[ \"$SECRETS_AD_SECRETS\" = \"/bin/sh $SECRETS_HOME/.bashrc\" ] || \
(echo "Failed: $SECRETS_HOME $SECRETS_AD_SECRETS"; exit 1)" > "$OUT"'''

gentest(
    name = "test_tildes_replaced",
    outs = ["test_tildes_replaced.sh"],
    cmd = tildes_replaced_test,
    no_test_output = True,
    secrets = {
        "home": ["~"],
        "ad_secrets": [
            "/bin/sh",
            "~/.bashrc",
        ],
    },
    test_cmd = "$TEST",
)
