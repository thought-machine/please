# Tests for Java Library including resources

subinclude("//build_defs:plz_e2e_test")

java_library(
    name = "issue_877",
    srcs = [
        "MyClass.java",
    ],
    resources = [
        "some.txt",
        # TODO(macripps): Add this once we work out better quoting support (see build_env.go).
        # "file with space.txt",
    ],
    # Ensure resources end up at root of jar
    resources_root = "test/java_rules/issue_877",
)

java_binary(
    name = "issue_877_main",
    main_class = "MyClass",
    deps = [
        ":issue_877",
    ],
)

plz_e2e_test(
    name = "issue_877_test",
    cmd = "unzip -l test/java_rules/issue_877/issue_877.jar",
    data = [
        ":issue_877",
    ],
    labels = [
        "java",
    ],
    expect_output_contains = "some.txt",
)
