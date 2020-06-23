if CONFIG.OS == 'linux' or CONFIG.OS == 'darwin':
    java_toolchain(
        name = "java14",
        sdk_url = {
            "linux": "https://download.java.net/java/GA/jdk14/076bab302c7b4508975440c56f6cc26a/36/GPL/openjdk-14_linux-x64_bin.tar.gz",
            "darwin": "https://download.java.net/java/GA/jdk14.0.1/664493ef4a6946b186ff29eb326336a2/7/GPL/openjdk-14.0.1_osx-x64_bin.tar.gz",
        },
    )

    java_toolchain(
        name = "java16",
        sdk_url = {
            'linux': "https://download.java.net/java/early_access/jdk16/2/GPL/openjdk-16-ea+2_linux-x64_bin.tar.gz",
            'darwin': "https://download.java.net/java/early_access/jdk16/2/GPL/openjdk-16-ea+2_osx-x64_bin.tar.gz",
        },
    )

    java_test(
        name = "java14_test",
        srcs = ["VersionTest.java"],
        deps = ["//third_party/java:junit"],
        jvm_args = "-Dexpected_version=14" if CONFIG.OS == 'linux' else '-Dexpected_version=14.0.1',
        toolchain = ":java14",
    )

    java_test(
        name = "java16_test",
        srcs = ["VersionTest.java"],
        deps = ["//third_party/java:junit"],
        jvm_args = "-Dexpected_version=16-ea",
        toolchain = ":java16",
    )

