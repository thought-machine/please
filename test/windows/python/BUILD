# The subject of //test/windows:pex_test. It is built for windows_amd64 and run under Wine;
# nothing runs it here.
python_test(
    name = "pex_test",
    srcs = ["pex_test.py"],
    data = [":data"],
    labels = ["manual"],
    visibility = ["//test/windows:all"],
    deps = ["//third_party/python:six"],
)

# The test reads this at the path it has here, so the Wine rule has to place it itself: a
# python_test's own data doesn't come along when another rule depends on the test.
filegroup(
    name = "data",
    srcs = ["data.txt"],
    test_only = True,
    visibility = ["//test/windows:all"],
)

# Windows has no exec(), so the .pex preamble runs the interpreter as a child process and has to
# pass its exit status back itself. Getting that wrong would make every failing python_binary
# look like it succeeded.
python_binary(
    name = "exit_code",
    labels = ["manual"],
    main = "exit_code.py",
    test_only = True,
    visibility = ["//test/windows:all"],
)
