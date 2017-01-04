# By default we use this as a Java compiler server. Obviously it can't compile itself
# though so we must use the system compiler here.
package(javac_tool = 'javac')

java_library(
    name = 'compile',
    srcs = glob(['*.java']),
    deps = [
        '//src/build/proto:worker',
    ],
    visibility = ['//tools/javac_worker:all'],
)
