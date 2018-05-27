java_test(
    name = "please_coverage_class_loader_test",
    srcs = ["PleaseCoverageClassLoaderTest.java"],
    deps = [
        "//third_party/java:junit",
    ],
)

java_test(
    name = "resources_root_test",
    srcs = ["ResourcesRootTest.java"],
    deps = [
        ":logback_test_xml",
        "//third_party/java:junit",
        "//third_party/java:logback-classic",
    ],
)

java_library(
    name = "logback_test_xml",
    resources = ["test_data/logback-test.xml"],
    resources_root = "test_data",
)
