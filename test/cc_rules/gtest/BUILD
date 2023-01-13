subinclude("///cc//build_defs:cc")

package(cc = {"test_main": "///third_party/cc/gtest//:gtest"})

cc_test(
    name = "gtest_test",
    srcs = ["gtest_test.cc"],
    deps = [
        "//test/cc_rules:lib2",
    ],
)
