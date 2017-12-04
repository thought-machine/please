package(python_test_runner = 'pytest')

python_library(
    name = 'inc',
    srcs = ['inc.py'],
)

python_test(
    name = 'pytest_test',
    srcs = ['test_runner.py'],
    deps = [':inc'],
)
