java_binary(
    name = "jacoco_all",
    deps = [
        ":asm",
        ":asm-analysis",
        ":asm-commons",
        ":asm-tree",
        ":asm-util",
        ":jacoco.agent",
        ":jacoco.core",
        ":jacoco.report",
    ],
)

genrule(
    name = "jacoco",
    srcs = {
        "jar": [":jacoco_all"],
        "rules": ["jarjar_rules.txt"],
    },
    outs = ["jacoco_shaded.jar"],
    cmd = "$TOOL process $SRCS_RULES $SRCS_JAR $OUT",
    tools = ["//third_party/java/jarjar:bin"],
    visibility = ["PUBLIC"],
)

maven_jar(
    name = "jacoco.core",
    hash = "5db0da8cb8905612df850d22c378912ead06c88f",
    id = "org.jacoco:org.jacoco.core:0.8.1",
)

maven_jar(
    name = "jacoco.report",
    hash = "be277d38f723a83e79304c9820df7655f8945d30",
    id = "org.jacoco:org.jacoco.report:0.8.1",
)

maven_jar(
    name = "jacoco.agent",
    hash = "7973f700eb8add2564871d1ed6d0ffad4e0ad9df",
    id = "org.jacoco:org.jacoco.agent:0.8.1",
)

maven_jar(
    name = "asm",
    hash = "sha1: 079196a08a86094763068ac621ba90c73c2efa34",
    id = "org.ow2.asm:asm:6.0",
)

maven_jar(
    name = "asm-commons",
    hash = "sha1: 079196a08a86094763068ac621ba90c73c2efa34",
    id = "org.ow2.asm:asm-commons:6.0",
)

maven_jar(
    name = "asm-tree",
    hash = "sha1: 079196a08a86094763068ac621ba90c73c2efa34",
    id = "org.ow2.asm:asm-tree:6.0",
)

maven_jar(
    name = "asm-analysis",
    hash = "sha1: 079196a08a86094763068ac621ba90c73c2efa34",
    id = "org.ow2.asm:asm-analysis:6.0",
)

maven_jar(
    name = "asm-util",
    hash = "sha1: 079196a08a86094763068ac621ba90c73c2efa34",
    id = "org.ow2.asm:asm-util:6.0",
)
