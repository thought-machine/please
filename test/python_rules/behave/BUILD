package(python_test_runner = "behave")

python_test(
    name = "behave_test",
    srcs = glob([
        "features/*",
    ]),
    flags = "-D test_value='this value should show up in my context' -D all_features=True",
)
