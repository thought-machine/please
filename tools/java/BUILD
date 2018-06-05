# This file defines prebuilt versions of the Java tools which can be used locally.
# That makes Java a soft dependency at build time - the bootstrap script takes care of which
# to use.

VER = "13.0.0-beta7"

# This is a prebuilt version of the JUnit runner which makes Java a soft dependency
# at build time (the bootstrap script takes care of which of these two to use).
remote_file(
    name = "please",
    hashes = ["e83b735374e43da0d9429b75065f89bb52b1f360"],
    url = "https://get.please.build/linux_amd64/%s/please_%s.tar.gz" % (VER, VER),
)

genrule(
    name = "junit_runner_prebuilt",
    srcs = [":please"],
    outs = ["junit_runner.jar"],
    cmd = "tar -xzf $SRCS please/junit_runner.jar && mv please/* .",
)

genrule(
    name = "javac_worker_prebuilt",
    srcs = [":please"],
    outs = ["javac_worker"],
    binary = True,
    cmd = "tar -xzf $SRCS please/javac_worker && mv please/* .",
)

# These rules switch between the prebuilt versions and the locally built ones based
# on a build config flag.
filegroup(
    name = "junit_runner",
    srcs = ["//tools/junit_runner" if CONFIG.get("BUILD_JAVA") else ":junit_runner_prebuilt"],
    visibility = ["PUBLIC"],
)

filegroup(
    name = "javac_worker",
    srcs = ["//tools/javac_worker" if CONFIG.get("BUILD_JAVA") else ":javac_worker_prebuilt"],
    binary = True,
    visibility = ["PUBLIC"],
)
