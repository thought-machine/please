subinclude("//test/build_defs")

genrule(
    name = "record_arch",
    outs = ["arch.txt"],
    cmd = 'echo ${OS}_${ARCH} > "$OUT"',
)

sh_test(
    name = "test_arch",
    src = "test_arch.sh",
    data = [":record_arch"],
    labels = ["manual"],
)

sh_test(
    name = "auto_arch_test",
    src = "test_arch.sh",
    data = ["///test_x86:record_arch"],
)

plz_e2e_test(
    name = "cross_compile_test",
    cmd = "plz test -a test_x86 //test/cross_compile:test_arch",
)

go_binary(
    name = "bin",
    srcs = ["main.go"],
    deps = ["//test/cross_compile/lib"],
)

sh_test(
    name = "bin_test",
    src = "bin_test.sh",
    data = [":bin"],
)

sh_test(
    name = "cross_compiled_bin_test",
    src = "cross_compiled_bin_test.sh",
    data = [":bin"],
    labels = ["manual"],
)

# N.B. Go can cross-compile to any platform it supports without needing a toolchain installed,
#      so unlike the C rules this used to use, this needs nothing set up to run.
plz_e2e_test(
    name = "cross_compile_binary_test",
    cmd = "plz test -a linux_arm64 //test/cross_compile:cross_compiled_bin_test",
)

# Test for #582
genrule(
    name = "location",
    srcs = [":record_arch"],
    outs = ["arch_location.txt"],
    cmd = "echo '$(location :record_arch)' > \"$OUT\"",
)

sh_test(
    name = "location_test",
    src = "location_test.sh",
    data = [":location"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "location_cross_compile_test",
    cmd = "plz test -a linux_arm64 //test/cross_compile:location_test",
)

config_setting(
    name = "linux_x86",
    values = {
        "os": "linux",
        "cpu": "x86",
    },
)

genrule(
    name = "wibble",
    outs = ["wibble.txt"],
    cmd = 'echo wibble > "$OUT"',
)

genrule(
    name = "wobble",
    outs = ["wobble.txt"],
    cmd = 'echo wobble > "$OUT"',
)

genrule(
    name = "select",
    srcs = select({
        "//test/select:linux_x86": [":wibble"],
        "default": [":wobble"],
    }),
    outs = ["select.txt"],
    cmd = "cat $SRCS > \"$OUT\"",
)

sh_test(
    name = "select_test",
    src = "select_test.sh",
    data = [":select"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "select_cross_compile_test",
    cmd = "plz test -a linux_x86 //test/cross_compile:select_test",
)
