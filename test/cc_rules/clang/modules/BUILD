# Tests on cc_module.
subinclude("///cc//build_defs:cc")

package(
    cc = {
        "cpp_tool": "clang++",
        "default_dbg_cppflags": "--std=c++20 -g3 -pipe -DDEBUG -Wall -Werror -Wno-unused-command-line-argument -stdlib=libstdc++",
        "default_opt_cppflags": "--std=c++20 -O3 -pipe -DNDEBUG -Wall -Werror -Wno-unused-command-line-argument -stdlib=libstdc++",
    }
)

cc_module(
    name = "hello",
    srcs = ["hello.cc"],
    interfaces = ["hello.cppm"],
)

cc_binary(
    name = "modules_test",
    srcs = ["main.cc"],
    deps = [":hello"],
)

cc_module(
    name = "f1",
    srcs = ["f1.cc"],
    interfaces = ["f1.cppm"],
    deps = [":q1"],
)

cc_module(
    name = "q1",
    srcs = ["q1.cc"],
    interfaces = ["q1.cppm"],
    deps = [":f1"],
)

cc_test(
    name = "interdependent_modules_test",
    srcs = ["interdependent_modules_test.cc"],
    labels = [
        "clang",
        "cc_module",
    ],
    deps = [
        ":f1",
        ":q1",
    ],
)
