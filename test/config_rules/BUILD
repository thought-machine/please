# This is a (fairly contrived) example of using select().
python_test(
    name = 'config_test',
    srcs = ['config_test.py'],
    flags = select({
        '//test/config_rules/config:x86_64': '--arch=x86_64',
        '//test/config_rules/config:x86': '--arch=x86',
    }),
)
