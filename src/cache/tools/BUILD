go_library(
    name = 'tools',
    # We need to use things from syscall that vary by platform.
    srcs = ['tools_linux.go' if (CONFIG.OS == 'linux') else 'tools.go'],
    visibility = ['//src/cache/...'],
)
