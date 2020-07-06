if CONFIG.OS == "linux" or CONFIG.OS == "darwin":
    java_toolchain(
        name = "java8",
        jdk_url = {
            "linux": "https://corretto.aws/downloads/latest/amazon-corretto-8-x64-linux-jdk.tar.gz",
            "darwin": "https://corretto.aws/downloads/latest/amazon-corretto-8-x64-macos-jdk.tar.gz",
        },
    )

    java_toolchain(
        name = "java11",
        jdk_url = {
            "linux": "https://corretto.aws/downloads/latest/amazon-corretto-11-x64-linux-jdk.tar.gz",
            "darwin": "https://corretto.aws/downloads/latest/amazon-corretto-11-x64-macos-jdk.tar.gz",
        },
    )

    java_toolchain(
        name = "java14",
        jdk_url = {
            "linux": "https://download.java.net/java/GA/jdk14/076bab302c7b4508975440c56f6cc26a/36/GPL/openjdk-14_linux-x64_bin.tar.gz",
            "darwin": "https://download.java.net/java/GA/jdk14.0.1/664493ef4a6946b186ff29eb326336a2/7/GPL/openjdk-14.0.1_osx-x64_bin.tar.gz",
        },
    )

    java_test(
        name = "java8_test",
        srcs = ["VersionTest.java"],
        jvm_args = "-Dexpected_version=1.8.0_252",
        toolchain = ":java8",
        deps = ["//third_party/java:junit"],
    )

    java_test(
        name = "java11_test",
        srcs = ["VersionTest.java"],
        jvm_args = "-Dexpected_version=11.0.7",
        toolchain = ":java11",
        deps = ["//third_party/java:junit"],
    )

    java_test(
        name = "java14_test",
        srcs = ["VersionTest.java"],
        jvm_args = "-Dexpected_version=14" if CONFIG.OS == "linux" else "-Dexpected_version=14.0.1",
        toolchain = ":java14",
        deps = ["//third_party/java:junit"],
    )
