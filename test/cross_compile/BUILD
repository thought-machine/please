subinclude("//test/build_defs", "///cc//build_defs:c")

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

c_binary(
    name = "bin",
    srcs = ["main.c"],
    deps = ["//test/cross_compile/lib"],
)

sh_test(
    name = "bin_test",
    src = "bin_test.sh",
    data = [":bin"],
    labels = ["manual"],
)

plz_e2e_test(
    name = "x86_test",
    cmd = "plz test -a linux_x86 //test/cross_compile:bin_test",
    labels = ["x86"],
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
    cmd = "plz test -a linux_x86 //test/cross_compile:location_test",
    labels = ["x86"],
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
    labels = ["x86"],
)
