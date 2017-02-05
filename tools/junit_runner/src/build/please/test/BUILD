java_library(
    name = 'junit_runner',
    srcs = glob(['*.java'], excludes = ['*Test.java']),
    exported_deps = [
        '//third_party/java:hamcrest',
        '//third_party/java:junit',
        '//third_party/java/jacoco',
    ],
    visibility = ['//tools/junit_runner/...'],
)

java_test(
    name = 'junit_runner_test',
    srcs = ['PleaseTestRunnerTest.java'],
    deps = [
        ':junit_runner',
        '//third_party/java:junit',
    ],
)

java_test(
    name = 'junit_runner_parameterized_test',
    srcs = ['PleaseTestRunnerParameterizedTest.java'],
    deps = [
        ':junit_runner',
        '//third_party/java:junit',
    ],
)

java_test(
    name = 'please_coverage_class_loader_test',
    srcs = ['PleaseCoverageClassLoaderTest.java'],
    deps = [
        ':junit_runner',
        '//third_party/java:junit',
    ],
)

java_test(
    name = 'resources_root_test',
    srcs = ['ResourcesRootTest.java'],
    deps = [
        ':junit_runner',
        ':logback_test_xml',
        '//third_party/java:logback-classic',
    ],
)

java_test(
    name = 'test_coverage_test',
    srcs = ['TestCoverageTest.java'],
    deps = [
        ':junit_runner',
    ],
)

java_library(
    name = 'logback_test_xml',
    resources = ['test_data/logback-test.xml'],
    resources_root = 'test_data',
)
