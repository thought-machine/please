# The plugin sources and helper tools the offline Windows release bundles.
#
# Nothing here is in git. tools/misc/vendor_plugins.sh generates it from the four plugin
# checkouts, and .gitignore covers the results; this file only describes them so that Please
# hashes their contents the way it hashes any other source.
#
# The whole package is a workaround for the plugin branches being unpublished, and goes away
# when they land. See docs/design/windows/08-offline-release.md, and the removal checklist in
# 07-state-of-play.md.
#
# Guarded so that an ordinary build, which has vendored nothing, still parses. The srcs are
# listed rather than globbed for the opposite reason: half a vendoring should fail loudly
# rather than ship a release quietly missing a plugin.
if CONFIG.get("BUNDLED_PLUGINS"):
    filegroup(
        name = "bundled",
        srcs = [
            "please_cc.exe",
            "please_go.exe",
            "please_pex.exe",
            "plugin_cc-rules.zip",
            "plugin_go-rules.zip",
            "plugin_python-rules.zip",
            "plugin_revisions.txt",
            "plugin_shell-rules.zip",
        ],
        binary = True,
        visibility = ["//package:all"],
    )
