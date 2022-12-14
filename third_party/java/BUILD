package(default_visibility = ["PUBLIC"])

subinclude("///java//build_defs:java")

maven_jar(
    name = "junit",
    hash = "59721f0805e223d84b90677887d9ff567dc534d7c502ca903c0c2b17f05c116a",
    id = "junit:junit:4.12",
    deps = [
        ":hamcrest",
    ],
)

maven_jar(
    name = "hamcrest",
    hash = "4877670629ab96f34f5f90ab283125fcd9acb7e683e66319a68be6eb2cca60de",
    id = "org.hamcrest:hamcrest-all:1.3",
)

maven_jar(
    name = "gson",
    hash = "d3999291855de495c94c743761b8ab5176cfeabe281a5ab0d8e8d45326fd703e",
    id = "com.google.code.gson:gson:2.8.9",
)

# These are shipped as components of Please, but for easier management the code lives in
# https://github.com/thought-machine/please-java
# As a result the versions are no longer directly connected to the ones in this repo.
maven_jar(
    name = "junit_runner",
    hashes = ["2b4af73adbba86f8c8a16def4d369682ba7de83147cfac09da549aba1ddf5f80"],
    id = "build.please:junit-runner:13.4.0",
    deps = [
        ":hamcrest",
        ":jacoco_shaded",
        ":junit",
    ],
)

maven_jar(
    name = "jacoco_shaded",
    hashes = ["9927bc9184d434c8ed0ed7ba46257df36e31dbd6bd224845d7a02d9e0894da4c"],
    id = "build.please:jacoco-shaded:0.8.4",
)

maven_jar(
    name = "javac_worker",
    hashes = ["63010a359fbac75beeac5f15f25b0fd489f1af8f7673ff8d170cc062eb1c699f"],
    id = "build.please:javac-worker:13.4.0",
    deps = [":gson"],
)
