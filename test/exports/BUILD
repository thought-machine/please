build_rule(
    name = "foo",
    outs = ["foo"],
    cmd = "touch $OUT",
    exports = [":bar"],
)

build_rule(
    name = "bar",
    outs = ["bar"],
    cmd = "touch $OUT",
    deps = [":foo"]
)

build_rule(
    name = "baz",
    outs = ["baz"],
    cmd = "touch $OUT",
    deps = [":foo"],
)