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
        name = "toolchain",
        hashes = [
            "bcd6c66f3c8a8b78c2caf7e0006be5598c365a2245120b6e8af9096d7c495c34",
            "a357054504bd55cceec52a9468834da5d12448e2b7779b6a1f24367d20454a33",
        ],
        jdk_url = {
            "linux": "https://corretto.aws/downloads/latest/amazon-corretto-11-x64-linux-jdk.tar.gz",
            "darwin": "https://corretto.aws/downloads/latest/amazon-corretto-11-x64-macos-jdk.tar.gz",
        },
    )
