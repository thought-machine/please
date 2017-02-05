package(default_visibility = ['PUBLIC'])

maven_jar(
    name = 'junit',
    id = 'junit:junit:4.12',
    hash = 'sha1: a791201ac8a3d2a251045a52e264de01343ad2df',
)

maven_jar(
    name = 'hamcrest',
    id = 'org.hamcrest:hamcrest-all:1.3',
    hash = 'sha1: 6e72502584462b2d35a7a6a18fc5541c412f08cc',
)

maven_jar(
    name = 'logback-classic',
    id = 'ch.qos.logback:logback-classic:1.1.3',
    hash = 'sha1: 245ace75b7a18931f103a79e74d53c66f1f90a9e',
    test_only = True,  # Only used in one test.
    deps = [
        ':logback-core',
        ':slf4j-api',
    ],
)

maven_jar(
    name = 'logback-core',
    id = 'ch.qos.logback:logback-core:1.1.3',
    hash = 'sha1: 53732a6ab1f058ad76808646519801e30cc4a780',
    test_only = True,
)

maven_jar(
    name = 'slf4j-api',
    id = 'org.slf4j:slf4j-api:1.7.10',
    hash = 'sha1: 0ff5ea6b9f644c97ef1de83eb75e7cc304103549',
    test_only = True,
)

maven_jar(
    name = 'protobuf',
    id = 'com.google.protobuf:protobuf-java:3.0.0',
    hash = '88d92d14d257f10971fbb734ffcdd290817045c0',
    sources = False,
)
