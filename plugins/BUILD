# Each plugin is pinned here as an archive download.
#
# To develop changes to one, put its checkout path in .plzconfig.local:
#
#     [buildconfig]
#     go-rules-path = /home/peter/code/go-rules
#
# and that directory is used in place of the download. The subrepo() builtin is called
# directly rather than through local_repository, which omits plugin = True and would register
# the subrepo as plugins/go rather than go, so that ///go//... never resolves.
PLUGINS = [
    ("go", "go-rules", "v1.31.1"),
    ("cc", "cc-rules", "v0.7.3"),
    ("shell", "shell-rules", "v0.2.1"),
    ("python", "python-rules", "v2.0.2"),
]

for name, plugin, revision in PLUGINS:
    local = CONFIG.get(plugin.replace("-", "_").upper() + "_PATH")
    if local:
        subrepo(
            name = name,
            path = local,
            plugin = True,
        )
    else:
        plugin_repo(
            name = name,
            plugin = plugin,
            revision = revision,
        )
