python_test(
    name = "isolated_custom_runner_test",
    srcs = ["custom_runner_test.py"],
    deps = [
        "//third_party/python:coverage",
        "//third_party/python:pytest",
    ],
    test_runner = "test.python_rules.custom_runner.runner.run",
)
