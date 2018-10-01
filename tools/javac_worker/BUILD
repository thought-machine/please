subinclude("//build_defs:maven_library")

subinclude("//build_defs:version")

# The rest of the repo uses this tool as a compiler worker, but it obviously
# can't compile itself.
package(javac_tool = "javac")

maven_library(
    name = "javac-worker",
    srcs = glob(["src/**/*.java"]),
    pom = "pom.xml",
    version = VERSION,
    deps = ["//third_party/java:gson"],
)

java_binary(
    name = "javac_worker",
    out = "javac_worker",
    main_class = "build.please.compile.JavaCompiler",
    self_executable = True,
    visibility = ["PUBLIC"],
    deps = [":javac-worker"],
)
