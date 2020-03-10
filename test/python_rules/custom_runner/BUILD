package(
    python_test_runner = "test.python_rules.custom_runner.runner.run",
    python_test_runner_bootstrap = "//test/python_rules/custom_runner:runner",
)

python_library(
    name = "runner",
    srcs = ["runner.py"],
    deps = ["//third_party/python:pytest"],
)

python_test(
    name = "custom_runner_test",
    srcs = ["custom_runner_test.py"],
)
