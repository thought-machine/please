# Test for java_library rules using src_dir
java_library(
    name = 'src_dir_lib',
    src_dir = 'src',
)

java_test(
    name = 'src_dir_test',
    srcs = ['test/build/please/java/test/SrcDirTest.java'],
    deps = [
        ':src_dir_lib',
        '//third_party/java:junit',
    ],
)



# Test for maven classifier
maven_jar(
    name = 'logback-core-tests',
    id = 'ch.qos.logback:logback-core:1.1.3',
    classifier = 'tests',
    classifier_sources_override = 'test',
    hash = '',
    test_only = True,
)

java_test(
    name = 'mvn_classifier_test',
    srcs = ['MvnClassifierTest.java'],
    deps = [
         ':logback-core-tests',
         '//third_party/java:junit',
    ],
)