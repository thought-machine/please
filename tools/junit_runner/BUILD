java_binary(
    name = "junit_runner_bin",
    main_class = "build.please.main.TestMain",
    visibility = ["PUBLIC"],
    deps = [
        "//tools/junit_runner/src/build/please/main:junit_runner",
    ],
)

genrule(
    name = "junit_runner",
    srcs = {
        "jar": [":junit_runner_bin"],
        "rules": ["jarjar_rules.txt"],
    },
    outs = ["junit_runner.jar"],
    cmd = "$TOOL process $SRCS_RULES $SRCS_JAR $OUT",
    tools = ["//third_party/java/jarjar:bin"],
    visibility = ["PUBLIC"],
)
