# This is a (fairly contrived) example of using select().
python_test(
    name = "config_test",
    srcs = ["config_test.py"],
    flags = select({
        "//test/config_rules/config:x86_64": "--word_size=64",
        "//test/config_rules/config:arm64": "--word_size=64",
        "//test/config_rules/config:x86": "--word_size=32",
    }),
)
