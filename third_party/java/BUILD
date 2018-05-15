package(default_visibility = ["PUBLIC"])

maven_jar(
    name = "junit",
    hash = "sha1: a791201ac8a3d2a251045a52e264de01343ad2df",
    id = "junit:junit:4.12",
)

maven_jar(
    name = "hamcrest",
    hash = "sha1: 6e72502584462b2d35a7a6a18fc5541c412f08cc",
    id = "org.hamcrest:hamcrest-all:1.3",
)

maven_jar(
    name = "logback-classic",
    hash = "sha1: 245ace75b7a18931f103a79e74d53c66f1f90a9e",
    id = "ch.qos.logback:logback-classic:1.1.3",
    test_only = True,  # Only used in one test.
    deps = [
        ":logback-core",
        ":slf4j-api",
    ],
)

maven_jar(
    name = "logback-core",
    hash = "sha1: 53732a6ab1f058ad76808646519801e30cc4a780",
    id = "ch.qos.logback:logback-core:1.1.3",
    test_only = True,
)

maven_jar(
    name = "slf4j-api",
    hash = "sha1: 0ff5ea6b9f644c97ef1de83eb75e7cc304103549",
    id = "org.slf4j:slf4j-api:1.7.10",
    test_only = True,
)

maven_jar(
    name = "protobuf",
    hash = "sha1: ea1115a7983c2c23b7d90ccef4a83675f4e3df9c",
    id = "com.google.protobuf:protobuf-java:3.4.0",
)

maven_jar(
    name = "gson",
    hash = "sha1: 19b10af44e08894efb629d90115b0c391eaaf55a",
    id = "com.google.code.gson:gson:2.8.4",
)
