package(python_test_runner = "pytest")

python_library(
    name = "inc",
    srcs = ["inc.py"],
)

python_library(
    name = "relative_imports",
    srcs = glob(["relative_imports/*"]),
)

python_test(
    name = "test_relative_imports",
    srcs = ["test_relative_imports.py"],
    deps = [
        ":relative_imports",
    ],
)

python_test(
    name = "pytest_test",
    srcs = ["test_runner.py"],
    deps = [":inc"],
)
