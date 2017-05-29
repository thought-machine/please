# Test on using secrets. Obviously it can't test any real secrets
# but it proves the system works.
gentest(
    name = 'secrets_test',
    outs = ['secrets_test.sh'],
    cmd = 'echo "#!/bin/sh\nexec $SECRETS" > $OUT',
    no_test_output = True,
    secrets = ['/bin/true'],
    test_cmd = '$TEST',
)
