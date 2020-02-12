gentest(
    name = "failing_test",
    test_cmd = "false",
    labels = ["manual"],
)

gentest(
    name = "passing_test",
    test_cmd = "true",
    no_test_output = True,
    deps = [":failing_test"],
)
