subinclude("///java//build_defs:java")

package(bazel_compatibility = True)

if is_platform(os = "linux") or is_platform(os = "darwin"):
    please_java_toolchain(
        name = "java8",
        jdk_url = {
            "linux": "https://corretto.aws/downloads/resources/8.262.10.1/amazon-corretto-8.262.10.1-linux-x64.tar.gz",
            "darwin": "https://corretto.aws/downloads/resources/8.262.10.1/amazon-corretto-8.262.10.1-macosx-x64.tar.gz",
        },
    )

    please_java_toolchain(
        name = "java11",
        jdk_url = {
            "linux": "https://corretto.aws/downloads/resources/11.0.8.10.1/amazon-corretto-11.0.8.10.1-linux-x64.tar.gz",
            "darwin": "https://corretto.aws/downloads/resources/11.0.8.10.1/amazon-corretto-11.0.8.10.1-macosx-x64.tar.gz",
        },
    )

    please_java_toolchain(
        name = "java14",
        jdk_url = {
            "linux": "https://download.java.net/java/GA/jdk14/076bab302c7b4508975440c56f6cc26a/36/GPL/openjdk-14_linux-x64_bin.tar.gz",
            "darwin": "https://download.java.net/java/GA/jdk14.0.1/664493ef4a6946b186ff29eb326336a2/7/GPL/openjdk-14.0.1_osx-x64_bin.tar.gz",
        },
    )

    java_test(
        name = "java8_test",
        srcs = ["VersionTest.java"],
        jvm_args = "-Dexpected_version=1.8.0_262",
        toolchain = ":java8",
        deps = ["//third_party/java:junit"],
    )

    java_test(
        name = "java11_test",
        srcs = ["VersionTest.java"],
        jvm_args = "-Dexpected_version=11.0.8",
        toolchain = ":java11",
        deps = ["//third_party/java:junit"],
    )

    java_test(
        name = "java14_test",
        srcs = ["VersionTest.java"],
        jvm_args = "-Dexpected_version=14" if is_platform(os = "linux") else "-Dexpected_version=14.0.1",
        toolchain = ":java14",
        deps = ["//third_party/java:junit"],
    )
