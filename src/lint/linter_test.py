import unittest

from src.lint import linter


lint = lambda fn, suppress=None: list(linter.lint(fn, suppress))


class TestLinter(unittest.TestCase):

    def test_iteritems(self):
        """Test that the linter finds dict.iteritems calls correctly."""
        self.assertEqual([(2, linter.ITERITEMS_USED), (7, linter.ITERITEMS_USED)],
                         lint('src/lint/test_data/test_iteritems'))

    def test_itervalues(self):
        """Test that the linter finds dict.itervalues calls correctly."""
        self.assertEqual([(2, linter.ITERVALUES_USED)], lint('src/lint/test_data/test_itervalues'))

    def test_iterkeys(self):
        """Test that the linter finds dict.iterkeys calls correctly."""
        self.assertEqual([(2, linter.ITERKEYS_USED)], lint('src/lint/test_data/test_iterkeys'))

    def test_syntax_error(self):
        """Test that we handle a syntax error gracefully."""
        self.assertEqual([(4, linter.SYNTAX_ERROR)], lint('src/lint/test_data/test_syntax_error'))

    def test_suppressions(self):
        """Test errors are suppressed correctly."""
        self.assertEqual([(6, linter.ITERITEMS_USED), (9, linter.ITERKEYS_USED)],
                         lint('src/lint/test_data/test_suppressions'))

    def test_unsorted_iteration(self):
        """Test unsorted iteration of set() and dict()."""
        self.assertEqual([(1, linter.UNSORTED_SET_ITERATION), (3, linter.UNSORTED_DICT_ITERATION)],
                         lint('src/lint/test_data/test_unsorted_iteration'))
