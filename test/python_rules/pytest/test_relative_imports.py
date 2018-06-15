"""
The tests below test out the relative imports within a package
"""
from test.python_rules.pytest.relative_imports import greetings, name


def test_function_call():
    assert greetings() == 'hello world'
    assert 1 == 1


def test_name():
    assert name == 'TM'
