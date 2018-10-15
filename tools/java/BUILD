# These are shipped as components of Please, but for easier management the code lives in
# https://github.com/thought-machine/please-java
# The versions tend to lag a little bit behind this repo as a result.
maven_jar(
    name = "javac_worker",
    hashes = ["no"],
    id = "build.please:javac-worker:13.2.0",
    sources = False,
    visibility = ["PUBLIC"],
)

maven_jar(
    name = "junit-runner",
    hashes = ["no"],
    id = "build.please:junit-runner:13.2.0",
    sources = False,
)

maven_jar(
    name = "jacoco_shaded",
    hashes = ["no"],
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
    ],
)
