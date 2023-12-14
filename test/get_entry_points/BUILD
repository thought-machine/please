none = genrule(
    name = "none",
    outs = ["x"],
    cmd = "echo x > $OUTS",
)

one = genrule(
    name = "one",
    outs = {
        "y": ["y"],
        "z": ["z"],
    },
    cmd = "for i in $OUTS; do echo x > $i; done",
    entry_points = {
        "ep": "y",
    },
)

def assert_dict(l1, l2):
    if l1 != l2:
        fail(f"{l1} != {l2}")

assert_dict(
    {},
    get_entry_points(none),
)

assert_dict(
    {"ep": "y"},
    get_entry_points(one),
)
