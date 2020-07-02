java_binary(
    name = "javac_worker",
    out = "javac_worker",
    main_class = "build.please.compile.JavaCompiler",
    self_executable = True,
    visibility = ["PUBLIC"],
    deps = ["//third_party/java:javac_worker"],
)

java_binary(
    name = "junit_runner",
    main_class = "build.please.main.TestMain",
    visibility = ["PUBLIC"],
    deps = ["//third_party/java:junit_runner"],
)
if CONFIG.OS == "linux" or CONFIG.OS == "darwin":
    java_toolchain(
        "toolchain",
        jdk_url = {
            'linux': "https://corretto.aws/downloads/latest/amazon-corretto-11-x64-linux-jdk.tar.gz",
            'darwin': "https://corretto.aws/downloads/latest/amazon-corretto-11-x64-macos-jdk.tar.gz",
        },
    )