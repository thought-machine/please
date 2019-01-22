package(default_visibility = ["PUBLIC"])

maven_jar(
    name = "junit",
    hash = "a791201ac8a3d2a251045a52e264de01343ad2df",
    id = "junit:junit:4.12",
    deps = [
        ":hamcrest",
    ],
)

maven_jar(
    name = "hamcrest",
    hash = "6e72502584462b2d35a7a6a18fc5541c412f08cc",
    id = "org.hamcrest:hamcrest-all:1.3",
)

maven_jar(
    name = "gson",
    hash = "19b10af44e08894efb629d90115b0c391eaaf55a",
    id = "com.google.code.gson:gson:2.8.4",
)

# These are shipped as components of Please, but for easier management the code lives in
# https://github.com/thought-machine/please-java
# The versions tend to lag a little bit behind this repo as a result.
maven_jar(
    name = "junit_runner",
    hashes = ["409563f6d2de11ef5e9e88275e1da62e984e4ccb"],
    id = "build.please:junit-runner:13.3.5",
    deps = [
        ":hamcrest",
        ":jacoco_shaded",
        ":junit",
    ],
)

maven_jar(
    name = "jacoco_shaded",
    hashes = ["1bf517865fcb9740159d6b4de4f6fb7b69f24aa9"],
    id = "build.please:jacoco-shaded:0.8.1",
)

maven_jar(
    name = "javac_worker",
    hashes = ["46b9672f0adb195e610fc86a656fd70ef3359d07"],
    id = "build.please:javac-worker:13.1.10",
    deps = [":gson"],
)
