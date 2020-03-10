import importlib


def run_tests(args):
    """Runs tests using a custom test runner that is selected by the user."""
    runner, _, name = "__TEST_RUNNER__".rpartition('.')
    mod = importlib.import_module(runner)
    return getattr(mod, name)(TEST_NAMES, args)
