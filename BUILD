filegroup(
    name = "version",
    srcs = ["VERSION"],
    visibility = ["PUBLIC"],
)

# This illustrates how to define a subrepo, which is used in test rules.
http_archive(
    name = "pleasings",
    hashes = ["388baebf9381c619f13507915f16d0165a5dc13e"],
    strip_prefix = "pleasings-f0c549b375067802400699247106e4907de917c2",
    urls = ["https://github.com/thought-machine/pleasings/archive/f0c549b375067802400699247106e4907de917c2.zip"],
)
