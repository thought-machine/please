go_library(
    name = "processhtml",
    srcs = [
        "errors.go",
        "html_traverser.go",
        "process_html_file.go",
    ],
    visibility = ["//docs/tools/fusejs_list_builder/..."],
    deps = [
        "///third_party/go/github.com_k3a_html2text//:html2text",
        "///third_party/go/golang.org_x_net//html",
        "///third_party/go/golang.org_x_net//html/atom",
        "//docs/tools/fusejs_list_builder/fusejslist",
    ],
)
