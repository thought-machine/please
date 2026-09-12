# Each plugin is pinned here as an archive download.
#
# These point at forks rather than at please-build, because the Windows support in all four is
# on a branch that is not merged anywhere. Pinned to commit SHAs rather than to that branch: a
# branch archive changes whenever it is pushed to, which would silently move every build hash
# that reaches it and leave the cache serving something else. When the work lands upstream, the
# owner goes back to please-build and these become ordinary version tags.
PLUGINS = [
    ("go", "go-rules", "a707213"),
    ("cc", "cc-rules", "9a130e1"),
    ("shell", "shell-rules", "7ed07be"),
    ("python", "python-rules", "5cc7dc2"),
]

for name, plugin, revision in PLUGINS:
    plugin_repo(
        name = name,
        owner = "PeterNeiss",
        plugin = plugin,
        revision = revision,
    )
