"""Tests on the pytest runner framework."""

# This isn't normally necessary for pytest, but proves that we're using the correct runner
# (otherwise unittest will just not call any of the test functions and report success)
import pytest


def inc(x):
    return x + 1


def test_answer():
    """Extremely simple test, from their examples."""
    assert inc(3) == 4


def test_pytest_is_importable():
    """Slightly more useful test; if pytest can't be imported, this can't be working..."""
    assert pytest
