# These are shipped as components of Please, but for easier management the code lives in
# https://github.com/thought-machine/please-java
# The versions tend to lag a little bit behind this repo as a result.
maven_jar(
    name = "javac_worker",
    hashes = ["8de5165b87b77b242adffb01c7d683a008d90244"],
    id = "build.please:javac-worker:13.1.10",
    sources = False,
    visibility = ["PUBLIC"],
)

maven_jar(
    name = "junit-runner",
    hashes = ["c93b0246878dbf58a789190cefe57c47431fdad2"],
    id = "build.please:junit-runner:13.2.0",
    sources = False,
)

maven_jar(
    name = "jacoco_shaded",
    hashes = ["9424429513230b26fb3767fe66d324be8a1fa2c9"],
    id = "build.please:jacoco-shaded:0.8.1",
    sources = False,
)

java_binary(
    name = "junit_runner",
    main_class = "build.please.main.TestMain",
    visibility = ["PUBLIC"],
    deps = [
        ":jacoco_shaded",
        ":junit-runner",
        "//third_party/java:hamcrest",
        "//third_party/java:junit",
    ],
)
