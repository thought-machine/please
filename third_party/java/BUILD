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

maven_jars(
    name = 'guava',
    id = 'com.google.guava:guava:18.0',
    exclude = ['jsr305'],
    hashes = {
        'guava': 'sha1: 8908f57125132a4468ea9977705a83f61b94fa89',
    }
)

maven_jars(
    name = 'jacoco',
    id = 'org.jacoco:jacoco:0.7.5.201505241946',
    exclude = [
        'org.jacoco.ant',
        'org.jacoco.doc',
        'org.jacoco.examples',
        'asm-debug-all',
    ],
    hashes = {
        'org.jacoco.report': 'sha1: 1129a80f257387660125fde69c82591effa53601',
        'org.jacoco.agent': 'sha1: 3d9535a24f7536f643059d7bc41dafd3d99c96fb',
        'org.jacoco.core': 'sha1: ff91273abd22add0bdda7aea226fcfe0026d7c4c',
    },
    deps = [':asm'],
    deps_only = True,
)

maven_jar(
    name = 'asm',
    id = 'org.ow2.asm:asm-debug-all:5.0.4',
    hash = 'sha1: 079196a08a86094763068ac621ba90c73c2efa34',
)

maven_jar(
    name = 'logback-classic',
    id = 'ch.qos.logback:logback-classic:1.1.3',
    hash = 'sha1: 245ace75b7a18931f103a79e74d53c66f1f90a9e',
    deps = [
        ':logback-core',
        ':slf4j-api',
    ]
)

maven_jar(
    name = 'logback-core',
    id = 'ch.qos.logback:logback-core:1.1.3',
    hash = 'sha1: 53732a6ab1f058ad76808646519801e30cc4a780',
)

maven_jar(
    name = 'slf4j-api',
    id = 'org.slf4j:slf4j-api:1.7.10',
    hash = 'sha1: 0ff5ea6b9f644c97ef1de83eb75e7cc304103549',
)

maven_jar(
    name = 'protobuf',
    id = 'com.google.protobuf:protobuf-java:3.0.0-beta-1',
    hash = 'b832e95bd94c68a01c32985d99ca5bb74bb0e0dc',
)
