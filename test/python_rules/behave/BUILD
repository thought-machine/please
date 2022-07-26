package(
    python = {
        "test_runner": "behave",
        "testrunner_deps": "///python//third_party/python:behave_bootstrap",
    }
)

python_test(
    name = "behave_test",
    srcs = glob([
        "features/*",
    ]),
    flags = "-D test_value='this value should show up in my context' -D all_features=True",
)
