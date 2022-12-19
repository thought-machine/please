# Identical set of tests to the main cc ones, but using Clang and Gold instead.
# (Note that this only controls the tools we invoke directly, not what they
# might use underneath - ie. we are not passing -fuse-ld or similar).

# Must pass -stdlib so the STL implementation matches what we use for other deps (i.e. unittest)
subinclude("//test/build_defs:cc_embed_binary", "///cc//build_defs:cc")

package(
    cc = {
        "cpp_tool": "clang++",
        "default_dbg_cppflags": "--std=c++11 -g3 -DDEBUG -Wall -Wextra -Werror -Wno-unused-parameter -stdlib=libstdc++",
        "default_namespace": "plz",
        "default_opt_cppflags": "--std=c++11 -O2 -DNDEBUG -Wall -Wextra -Werror -Wno-unused-parameter -stdlib=libstdc++",
        "ld_tool": "gold",
    }
)

cc_embed_binary(
    name = "embedded_file_1",
    src = "embedded_file_1.txt",
)

genrule(
    name = "embedded_file_3_gen",
    outs = ["embedded_file_3.txt"],
    cmd = "echo \"testing message 3\" > \"$OUT\"",
)

c_embed_binary(
    name = "embedded_file_3",
    src = ":embedded_file_3_gen",
    deps = [":embedded_file_3_gen"],
)

cc_test(
    name = "embed_file_test",
    srcs = ["embed_file_test.cc"],
    labels = [
        "clang",
        "gold",
        "embed",
    ],
    deps = [
        ":embedded_file_1",
        ":embedded_file_3",
    ],
)

# This is a little chain of tests to exercise the cc_shared_object rule.
cc_library(
    name = "embedded_files",
    srcs = ["embedded_files.cc"],
    hdrs = ["embedded_files.h"],
    deps = [
        ":embedded_file_1",
        ":embedded_file_3",
    ],
)

cc_shared_object(
    name = "so_test",
    srcs = ["so_test.cc"],
    out = "so_test.so",
    linker_flags = ["-bundle -undefined dynamic_lookup"] if is_platform(os = "darwin") else [],
    pkg_config_cflags = ["python3"],
    deps = [
        ":embedded_files",
    ],
)

python_test(
    name = "shared_object_test",
    srcs = ["shared_object_test.py"],
    labels = [
        "cc",
        "clang",
        "gold",
        "py3_pkg_config",
        "embed",
    ],
    zip_safe = False,
    deps = [
        ":so_test",
    ],
)

# Make sure we have at least one working cc_binary rule.
cc_binary(
    name = "test_binary",
    srcs = ["test_binary.cc"],
    deps = [
        ":embedded_files",
    ],
)

gentest(
    name = "cc_binary_test",
    data = [":test_binary"],
    labels = [
        "cc",
        "clang",
        "gold",
        "embed",
    ],
    no_test_output = True,
    test_cmd = "$(exe :test_binary)",
)
