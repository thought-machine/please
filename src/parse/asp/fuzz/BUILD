genrule(
    name = "build",
    srcs = ["fuzz.go"],
    outs = ["fuzz.zip"],
    cmd = 'export GOPATH=$TMP_DIR:$TMP_DIR/third_party/go; "$TOOL" -o "$OUT" parse/asp/fuzz',
    labels = ["manual"],  # This is really quite slow to compile and has poor incrementality.
    needs_transitive_deps = True,
    requires = [
        "go",
        "go_src",
    ],
    tools = ["//third_party/go:go-fuzz-build"],
    deps = [
        "//rules",
        "//src/core",
        "//src/parse/asp",
        "//third_party/go:go-fuzz",
    ],
)
