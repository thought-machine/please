filegroup(
    name = "version",
    srcs = ["VERSION"],
    visibility = ["PUBLIC"],
)

new_http_archive(
    name = "gtest",
    build_file = "gtest.build",
    strip_prefix = "googletest-b4d4438df9479675a632b2f11125e57133822ece",
    urls = ["https://github.com/google/googletest/archive/b4d4438df9479675a632b2f11125e57133822ece.zip"],
)
