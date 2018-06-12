java_binary(
    name = "bin",
    main_class = "org.pantsbuild.jarjar.Main",
    self_executable = True,
    visibility = ["//third_party/java/jacoco:all"],
    deps = [":jarjar"],
)

maven_jar(
    name = "jarjar",
    hash = "0e91aeb9659631d5a0e1082ed94f1bcfa143760f",
    id = "org.pantsbuild:jarjar:1.6.5",
    deps = [
        ":ant",
        ":asm",
        ":asm-commons",
        ":maven-plugin-api",
    ],
)

maven_jar(
    name = "ant",
    hash = "fc6443190de26e6a5d6ae17dbde178ee234f0388",
    id = "org.apache.ant:ant:1.9.9",
    deps = [
        ":ant-launcher",
    ],
)

maven_jar(
    name = "ant-launcher",
    hash = "5506e780cd3f5384cc3d7d27c59a522c7f4809fa",
    id = "org.apache.ant:ant-launcher:1.9.9",
)

maven_jar(
    name = "asm",
    hash = "9b66ade83c8663b1d0d8b96c68735cd7819d5974",
    id = "org.ow2.asm:asm:6.0",
)

maven_jar(
    name = "asm-commons",
    hash = "974412964b72c4c7b0b02243827a6bdd6ad982ad",
    id = "org.ow2.asm:asm-commons:6.0",
    deps = [
        ":asm-tree",
    ],
)

maven_jar(
    name = "asm-tree",
    hash = "d77ccf739f87599e427d1efac3f2d7c5f76f9fd3",
    id = "org.ow2.asm:asm-tree:6.0",
    deps = [
        ":asm",
    ],
)

maven_jar(
    name = "maven-plugin-api",
    hash = "f2f4e2f7dbff6c068e41de1304b91453012b48f2",
    id = "org.apache.maven:maven-plugin-api:3.3.9",
    deps = [
        ":maven-artifact",
        ":maven-model",
        ":org.eclipse.sisu.plexus",
    ],
)

maven_jar(
    name = "maven-model",
    hash = "19821de6ea09d34dbd70eb14c6d9cd17189edabe",
    id = "org.apache.maven:maven-model:3.3.9",
    deps = [
        ":commons-lang3",
        ":plexus-utils",
    ],
)

maven_jar(
    name = "plexus-utils",
    hash = "8a7bd4a58cae1999fdbfdec514e873a0c27fcced",
    id = "org.codehaus.plexus:plexus-utils:3.0.17",
)

maven_jar(
    name = "commons-lang3",
    hash = "297582124e343dcfbea999d523c937d97301533a",
    id = "org.apache.commons:commons-lang3:3.7",
)

maven_jar(
    name = "maven-artifact",
    hash = "7a73faa5a030214a288792896588f85f44c4ca02",
    id = "org.apache.maven:maven-artifact:3.3.9",
    deps = [
        ":commons-lang3",
        ":plexus-utils",
    ],
)

maven_jar(
    name = "org.eclipse.sisu.plexus",
    hash = "6ddef4d45dd47b85daf55283b436e1f5d9426565",
    id = "org.eclipse.sisu:org.eclipse.sisu.plexus:0.3.3",
    deps = [
        ":cdi-api",
        ":guice",
        ":org.eclipse.sisu.inject",
        ":plexus-classworlds",
        ":plexus-component-annotations",
        ":plexus-utils",
    ],
)

maven_jar(
    name = "guice",
    hash = "2a70a2fbf10a6f3eb515615cae059a627bcabf5e",
    id = "com.google.inject:guice:3.0",
    deps = [
        ":aopalliance",
        ":cglib",
        ":javax.inject",
    ],
)

maven_jar(
    name = "javax.inject",
    hash = "18940315facff73d8e9604c13bd4f2ff0e3de411",
    id = "javax.inject:javax.inject:1",
)

maven_jar(
    name = "aopalliance",
    hash = "a5574ecdcf2c9f3d6924088a1374588178947392",
    id = "aopalliance:aopalliance:1.0",
)

maven_jar(
    name = "cglib",
    hash = "a8d4c16397c76cb74424ec221e588ec8bbd4094f",
    id = "org.sonatype.sisu.inject:cglib:2.2.1-v20090111",
    deps = [
        ":asm",
    ],
)

maven_jar(
    name = "cdi-api",
    hash = "e5c622479ec26e8171f586dd9d94b016d08c413b",
    id = "javax.enterprise:cdi-api:1.0",
    deps = [
        ":javax.inject",
        ":jsr250-api",
    ],
)

maven_jar(
    name = "jsr250-api",
    hash = "16112da000ef071ae5537353701adc68294b12fc",
    id = "javax.annotation:jsr250-api:1.0",
)

maven_jar(
    name = "org.eclipse.sisu.inject",
    hash = "342575ddaf98ddf9d34a5511d80b76e5490a34be",
    id = "org.eclipse.sisu:org.eclipse.sisu.inject:0.3.3",
    deps = [
        ":guice",
    ],
)

maven_jar(
    name = "plexus-component-annotations",
    hash = "56417a86928452b4c9b12710c248d2303e85d756",
    id = "org.codehaus.plexus:plexus-component-annotations:1.5.5",
)

maven_jar(
    name = "plexus-classworlds",
    hash = "020a982bd64d6454618b1747cd93e07878b64633",
    id = "org.codehaus.plexus:plexus-classworlds:2.5.2",
)
