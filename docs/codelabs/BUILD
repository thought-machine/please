subinclude("//docs/build_defs")

deps = []

for src in glob(["*.md"]):
    deps += [codelab(name = src.removesuffix(".md"), codelab_md = src)]

filegroup(
    name = "codelabs",
    srcs = deps + [
        "style.css",
        "codelab.js",
    ],
    visibility = ["//docs/..."],
)

go_binary(
    name = "codelab_templator",
    srcs = ["codelab_template.go"],
    deps = ["///third_party/go/github.com_peterebden_go-cli-init_v5//flags"],
)

genrule(
    name = "templated_index",
    srcs = {
        "TEMPLATE": ["template.html"],
        "LABS": glob(["*.md"]),
    },
    outs = ["codelabs.html"],
    cmd = "$TOOLS --template $SRCS_TEMPLATE $SRCS_LABS > $OUT",
    tools = [":codelab_templator"],
    visibility = ["//docs/..."],
)
