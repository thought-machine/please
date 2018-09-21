subinclude("//build_defs:version")

subinclude("//build_defs:signing")

genrule(
    name = "worker",
    srcs = ["//tools/javac_worker"],
    outs = [f"javac-worker-{VERSION}.jar"],
    cmd = 'cp $SRCS $OUT',
)

detached_signature(
    name = "worker_signed",
    src = "//tools/javac_worker",
    out = f"javac-worker-{VERSION}.jar.asc",
)

filegroup(
    name = "maven",
    srcs = [
        ":worker",
        ":worker_signed",
        "//tools/javac_worker:pom",
    ],
)
