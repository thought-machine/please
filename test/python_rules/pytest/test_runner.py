"""Tests on the pytest runner framework."""

# This isn't normally necessary for pytest, but proves that we're using the correct runner
# (otherwise unittest will just not call any of the test functions and report success)
import pytest


def test_answer():
    """Deceptively simple test, from their examples.

    In this case, it's really testing that the import works correctly, which is fairly important...
    """
    from test.python_rules.pytest.inc import inc
    assert inc(3) == 4


def test_pytest_is_importable():
    """Slightly more useful test; if pytest can't be imported, this can't be working..."""
    assert pytest

