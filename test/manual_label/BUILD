gentest(
    name = "failing_test",
    labels = ["manual"],
    test_cmd = "false",
)

gentest(
    name = "passing_test",
    no_test_output = True,
    test_cmd = "true",
    deps = [":failing_test"],
)
