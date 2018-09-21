java_binary(
    name = "javac_worker",
    out = "javac_worker",
    main_class = "build.please.compile.JavaCompiler",
    self_executable = True,
    visibility = ["PUBLIC"],
    deps = ["//tools/javac_worker/src/build/please/compile"],
)

# N.B. The artifact we upload is //tools/javac_worker/src/build/please/compile,
#      because the binary above contains its dependencies too.
genrule(
    name = "pom",
    srcs = {
        "pom": ["pom.xml"],
        "ver": ["//:version"],
    },
    outs = ["pom.xml"],
    cmd = "sed s/__VERSION__/`cat $SRCS_VER`/g $SRCS > $OUT",
    visibility = ["//package/..."],
)

filegroup(
    name = "maven_jar",
    srcs = ["//tools/javac_worker/src/build/please/compile"],
    visibility = ["//package/..."],
)
