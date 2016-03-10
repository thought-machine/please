# A set of build targets that test issue #43.
# Due to the nature of how we exercise the issue the test itself cannot be in this package.

python_library(
    name = 'cycle_1',
    deps = [':cycle_2'],
)

python_library(
    name = 'cycle_2',
    deps = [':cycle_1'],
)

gentest(
    name = 'cycle_test',
    test_cmd = 'true',
    no_test_output = True,
    deps = [':cycle_1', ':cycle_2'],
    labels = ['cycle'],
)
