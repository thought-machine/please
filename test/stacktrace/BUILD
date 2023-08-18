

expect = [
    {
        "filename": "test/stacktrace/BUILD",
        "subrepo": "",
        "package": "test/stacktrace",
        "function": "<lambda>"
    },
    {
        "filename": "test/stacktrace/BUILD",
        "subrepo": "",
        "package": "test/stacktrace",
        "function": "c"
    },
    {
        "filename": "test/stacktrace/BUILD",
        "subrepo": "",
        "package": "test/stacktrace",
        "function": "b"
    },
    {
        "filename": "test/stacktrace/BUILD",
        "subrepo": "",
        "package": "test/stacktrace",
        "function": "a"
    },
]


def d(res):
    return res

def c():
    x = lambda: stacktrace()
    return d(x())

def b():
    return c()

def a():
    return b()

trace = a()

if len(trace) != len(expect):
    fail(f"trace doesnt have expected lenght\n{expect}\n{trace}")

for i in range(len(expect)):
    if trace[i] != expect[i]:
        fail(f"trace is different than expected\n{expect}\n{trace}")



