plugins = {
    "python": "v1.2.0",
    "java": "v1.0.0",
    "go": "v1.1.4",
    "cc": "v1.0.0",
    "shell": "v1.0.0",
}

for plugin, tag in plugins.items():
    http_archive(
        name = f"{plugin}",
        urls = [f"https://github.com/please-build/{plugin}-rules/archive/refs/tags/{tag}.tar.gz"],
        visibility = ["PUBLIC"],
    )

#     genrule(
#         name = f"{plugin}_extract_build_defs",
#         srcs = [f":{plugin}"],
#         outs = [f"{plugin}_build_defs"],
#         cmd = [
#             "mkdir -p $OUT",
#             f"find $SRCS -type f -name '*.build_defs' -exec mv {{}} $OUT \;",
#         ],
#     )
# 
#     filegroup(
#         name = f"{plugin}_build_defs",
#         srcs = [f":{plugin}_extract_build_defs"],
#         visibility = ["PUBLIC"],
#     )

# go_binary(
#     name = "rules",
