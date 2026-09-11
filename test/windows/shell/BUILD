# The subject of //test/windows:sh_binary_test. Built for windows_amd64 and run under Wine;
# nothing builds these here, hence the manual labels.
sh_library(
    name = "lib",
    src = "lib.sh",
    labels = ["manual"],
)

# On Windows this comes out as greet.cmd: a batch preamble that unpacks the zip appended to it
# and hands the script to the bundled busybox. It sources the library out of the unpacked
# payload, takes an argument and exits non-zero, so the wrapper has to get all three right.
sh_binary(
    name = "greet",
    labels = ["manual"],
    main = "greet.sh",
    visibility = ["//test/windows:all"],
    deps = [":lib"],
)
