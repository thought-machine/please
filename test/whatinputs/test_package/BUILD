genrule(
    name = "target1",
    srcs = ["foo.txt"],
    outs = ["out.txt"],
    cmd = "cat $SRCS > $OUT",
)

genrule(
    name = "_target2",
    srcs = ["_target2#child"],
    outs = ["out2.txt"],
    cmd = "cat $SRCS > $OUT",
)

genrule(
    name = "_target2#child",
    srcs = ["bar.txt"],
    outs = ["out2_child.txt"],
    cmd = "cat $SRCS > $OUT",
)
