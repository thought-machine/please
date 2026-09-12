# Each plugin is pinned here as an archive download.
#
# These point at forks rather than at please-build, because the Windows support in all four is
# on a branch that is not merged anywhere. Pinned to commit SHAs rather than to that branch: a
# branch archive changes whenever it is pushed to, which would silently move every build hash
# that reaches it and leave the cache serving something else. When the work lands upstream, the
# owner goes back to please-build and these become ordinary version tags.
PLUGINS = [
    ("go", "go-rules", "abd06f1"),
    ("cc", "cc-rules", "90913bb"),
    ("shell", "shell-rules", "7ed07be"),
    ("python", "python-rules", "2edd835"),
]

for name, plugin, revision in PLUGINS:
    plugin_repo(
        name = name,
        owner = "PeterNeiss",
        plugin = plugin,
        revision = revision,
    )
