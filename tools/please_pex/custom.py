import importlib


def run_tests(args):
    """Runs tests using a custom test runner that is selected by the user."""
    runner = "__TEST_RUNNER__"
    mod_name, _, name = runner.rpartition('.')
    mod = importlib.import_module(mod_name)
    f = getattr(mod, name)
    if not callable(f):
        raise TypeError('Specified test runner %s is not callable, should be a function taking two arguments' % runner)
    return f(TEST_NAMES, args)
