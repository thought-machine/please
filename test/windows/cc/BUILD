# The subject of //test/windows:dll_test. Built for windows_amd64 and run under Wine; nothing
# builds these here, hence the manual labels.
subinclude("///cc//build_defs:cc")

cc_shared_object(
    name = "greeting",
    srcs = ["greeting.cpp"],
    hdrs = ["greeting.h"],
    labels = ["manual"],
    visibility = ["//test/windows:all"],
)

# -lgreeting resolves against the import library the shared object emits alongside the DLL,
# which is the part that did not exist before. The DLL itself is a run-time dependency: Windows
# has no rpath, so it has to be beside the binary, which is what the test rule arranges.
cc_binary(
    name = "hello",
    srcs = ["main.cpp"],
    hdrs = ["greeting.h"],
    labels = ["manual"],
    linker_flags = [
        "-L" + package_name(),
        "-lgreeting",
    ],
    visibility = ["//test/windows:all"],
    deps = [":greeting"],
)

# The subject of //test/windows:cc_test_test. cc_test was recorded as blocked on Windows,
# because UnitTest++ supposedly needs Win32 sources the plugin does not include - which is not
# true, upstream selects them. This is here to find out what, if anything, is actually wrong.
cc_test(
    name = "greeting_test",
    srcs = ["greeting_test.cpp"],
    hdrs = ["greeting.h"],
    labels = ["manual"],
    linker_flags = [
        "-L" + package_name(),
        "-lgreeting",
    ],
    visibility = ["//test/windows:all"],
    deps = [":greeting"],
)

# The MinGW C++ runtime, which a cross-built binary needs beside it unless it was linked
# statically. The cc plugin wraps linker_flags in -Wl, so -static-libstdc++ cannot be passed
# through it - those are driver flags, and ld rejects them - and target compiler_flags reach the
# compile step but not the link. So the DLLs travel with the test, the same way the greeting DLL
# does. A cc_binary doing less C++ gets away without them, which is why the DLL test never
# needed this.
#
# Taken from the toolchain rather than pinned: they have to match the compiler that built the
# binary, and -print-file-name is how the compiler says where its own runtime is.
genrule(
    name = "mingw_runtime",
    outs = [
        "libgcc_s_seh-1.dll",
        "libstdc++-6.dll",
    ],
    binary = True,
    cmd = 'for dll in $OUTS; do cp "$($TOOL -print-file-name=$(basename $dll))" "$dll"; done',
    labels = ["manual"],
    test_only = True,
    # The plugin decides this per platform now and leaves the config key unset, so name the
    # cross compiler directly, falling back to whatever the repo configured if it did.
    tools = [CONFIG.CC.CPP_TOOL or "x86_64-w64-mingw32-g++"],
    visibility = ["//test/windows:all"],
)
