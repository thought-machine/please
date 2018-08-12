filegroup(
    name = "version",
    srcs = ["VERSION"],
    visibility = ["PUBLIC"],
)

new_local_repository(
    name = "pleasings",
    path = "../pleasings",
)

http_archive(
     name = "gtest",
     urls = ["https://github.com/google/googletest/archive/b4d4438df9479675a632b2f11125e57133822ece.zip"],
     strip_prefix = "googletest-b4d4438df9479675a632b2f11125e57133822ece",
)
