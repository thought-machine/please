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
