java_library(
    name = "jacoco",
    exported_deps = [
        ":asm",
        ":jacoco.agent",
        ":jacoco.core",
        ":jacoco.report",
    ],
    visibility = ["PUBLIC"],
)

maven_jar(
    name = "jacoco.core",
    hash = "5db0da8cb8905612df850d22c378912ead06c88f",
    id = "org.jacoco:org.jacoco.core:0.7.7.201606060606",
)

maven_jar(
    name = "jacoco.report",
    hash = "be277d38f723a83e79304c9820df7655f8945d30",
    id = "org.jacoco:org.jacoco.report:0.7.7.201606060606",
)

maven_jar(
    name = "jacoco.agent",
    hash = "7973f700eb8add2564871d1ed6d0ffad4e0ad9df",
    id = "org.jacoco:org.jacoco.agent:0.7.7.201606060606",
)

maven_jar(
    name = "asm",
    hash = "sha1: 079196a08a86094763068ac621ba90c73c2efa34",
    id = "org.ow2.asm:asm-debug-all:5.0.4",
)
