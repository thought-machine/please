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
