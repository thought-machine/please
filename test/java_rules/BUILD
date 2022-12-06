subinclude("///java//build_defs:java")

# Test for java_library rules using src_dir
java_library(
    name = "src_dir_lib",
    src_dir = "src",
)

java_test(
    name = "src_dir_test",
    srcs = ["test/build/please/java/test/SrcDirTest.java"],
    deps = [
        ":src_dir_lib",
        "//third_party/java:junit",
    ],
)

# Test for maven classifier
maven_jar(
    name = "logback-core-tests",
    classifier = "tests",
    classifier_sources_override = "test",
    hash = "d3a2f216851a8056de75e1862fcba21287b548afbd8a0fb6c66c53feedeb3e47",
    id = "ch.qos.logback:logback-core:1.1.3",
    test_only = True,
)

java_test(
    name = "mvn_classifier_test",
    srcs = ["test/build/please/java/test/MvnClassifierTest.java"],
    deps = [
        ":logback-core-tests",
        "//third_party/java:junit",
    ],
)
