v_none = None

v_bool = False

v_int = 5

v_str = "hello"

v_range = range(1, 6)

v_list = [
    0,
    1,
    2,
]

v_dict = {
    "a": 0,
    "b": 1,
}

def func1():
    fail("func1 should not be executed")

lambda1 = lambda: fail("lambda1 should not be executed")

all_type = [
    None,
    bool,
    int,
    str,
    range,
    list,
    dict,
    callable,
]

matrix = [
    [
        v_none,
        None,
    ],
    [
        v_bool,
        bool,
    ],
    [
        v_int,
        int,
    ],
    [
        v_str,
        str,
    ],
    [
        v_range,
        range,
    ],
    [
        v_list,
        list,
    ],
    [
        v_dict,
        dict,
    ],
    [
        func1,
        callable,
    ],
    [
        lambda1,
        callable,
    ],
]

for ty in matrix:
    v = ty[0]
    realtype = ty[1]
    istypes = {str(t): isinstance(v, t) for t in all_type}

    realtypestr = str(realtype)
    for typestr, istype in istypes.items():
        if typestr == realtypestr and not istype:
            fail(f"{v} should be of type {typestr} -> {istypes}")
        elif realtype == bool and typestr == "<function int>":  # special bool case which is also int
            continue
        elif typestr != realtypestr and istype:
            fail(f"{v} should not be of type {typestr} -> {istypes}")
