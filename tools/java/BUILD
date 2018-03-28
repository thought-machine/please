# This file defines prebuilt versions of the Java tools which can be used locally.
# That makes Java a soft dependency at build time - the bootstrap script takes care of which
# to use.

VER = '12.0.0'

# This is a prebuilt version of the JUnit runner which makes Java a soft dependency
# at build time (the bootstrap script takes care of which of these two to use).
remote_file(
    name = 'please',
    hashes = ['81e0a907c169c137f6992c1a54fd0439f1e234a9'],
    url = 'https://github.com/thought-machine/please/releases/download/v%s/please_%s_linux_amd64.tar.gz' % (VER, VER),
)

genrule(
    name = 'junit_runner_prebuilt',
    srcs = [':please'],
    outs = ['junit_runner.jar'],
    cmd = 'tar -xzf $SRCS please/junit_runner.jar && mv please/* .',
)

genrule(
    name = 'javac_worker_prebuilt',
    srcs = [':please'],
    outs = ['javac_worker'],
    binary = True,
    cmd = 'tar -xzf $SRCS please/javac_worker && mv please/* .',
)

# These rules switch between the prebuilt versions and the locally built ones based
# on a build config flag.
filegroup(
    name = 'junit_runner',
    srcs = ['//tools/junit_runner' if CONFIG.get('BUILD_JAVA') else ':junit_runner_prebuilt'],
    visibility = ['PUBLIC'],
)

filegroup(
    name = 'javac_worker',
    srcs = ['//tools/javac_worker' if CONFIG.get('BUILD_JAVA') else ':javac_worker_prebuilt'],
    binary = True,
    visibility = ['PUBLIC'],
)
