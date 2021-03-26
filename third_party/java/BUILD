package(default_visibility = ["PUBLIC"])

maven_jar(
    name = "junit",
    hash = "9f43fea92033ad82bcad2ae44cec5c82abc9d6ee4b095cab921d11ead98bf2ff",
    id = "junit:junit:4.12",
    deps = [
        ":hamcrest",
    ],
)

maven_jar(
    name = "hamcrest",
    hash = "c53535c3d25b5bf0b00a324a5583c7dd2fed0fa6d1bbc622e2dec460c24faab3",
    id = "org.hamcrest:hamcrest-all:1.3",
)

maven_jar(
    name = "gson",
    hash = "32821f053a181c3568ceeeff571e54de9f68e1bca99f92e37cf966a5583b8637",
    id = "com.google.code.gson:gson:2.8.4",
)

# These are shipped as components of Please, but for easier management the code lives in
# https://github.com/thought-machine/please-java
# As a result the versions are no longer directly connected to the ones in this repo.
maven_jar(
    name = "junit_runner",
    hashes = ["59f642a5fc6addc429ec4cac1364fa1d45000525f43fbc2fbef35cf585cbadeb"],
    id = "build.please:junit-runner:13.4.0",
    deps = [
        ":hamcrest",
        ":jacoco_shaded",
        ":junit",
    ],
)

maven_jar(
    name = "jacoco_shaded",
    hashes = ["8739c76e681f900923b900c9df0ef75cf421d39cabb54650c4b9ad19b6a76d85"],
    id = "build.please:jacoco-shaded:0.8.4",
)

maven_jar(
    name = "javac_worker",
    hashes = ["3c49d7a4965aaf17fa409416df38b0ead2a0535d9a7e4dd75329557ef63ec964"],
    id = "build.please:javac-worker:13.4.0",
    deps = [":gson"],
)
