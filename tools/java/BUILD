subinclude("///java//build_defs:java")

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
            "dbbf98ca93b44a0c81df5a3a4f2cebf467ec5c30e28c359e26615ffbed0b454f",
            "f1d2d566c189075c5e543076df61b452f830d8893bbc49b6fb2057070ff97818",
        ],
        jdk_url = {
            "linux": "https://corretto.aws/downloads/resources/11.0.8.10.1/amazon-corretto-11.0.8.10.1-linux-x64.tar.gz",
            "darwin": "https://corretto.aws/downloads/resources/11.0.8.10.1/amazon-corretto-11.0.8.10.1-macosx-x64.tar.gz",
        },
    )
