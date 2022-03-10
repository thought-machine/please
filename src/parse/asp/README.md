ASP
===

This package implements a custom parser for BUILD files.

Asp is syntactically a subset of Python, with many of its more advanced or dynamic
features stripped out. Some of the notable differences are:
 * The `import`, `try`, `except`, `finally`, `class`, `global`, `nonlocal`,
   `while` and `async` keywords are not available. These are also prohibited from
   being used as identifiers in order to maintain compatibility.
 * The `assert` statement is supported, but it is not possible to catch any
   resulting error (since `try` and `except` don't exist).
 * List and dict comprehensions are supported, but not Python's more general
   generator expressions. Up to two 'for' clauses are permitted.
 * Most builtin functions are not available.
 * Dictionaries are supported, but can only be keyed by strings. They always
   iterate in a consistent order.
 * The only builtin types are `bool`, `int`, `str`, `list`, `dict` and functions.
   There are no `float`, `complex`, `set`, `frozenset` or `bytes` types.
 * Operators `+`, `<`, `>`, `%`, `and`, `or`, `in`, `not in`, `is`, `is not`,
   `==`, `>=`, `<=` and `!=` are supported in most appropriate cases. Other operators
   are not available.
 * String interpolation is available via `%` and f-strings. `format()` is also available
   but its implementation is incomplete and use is discouraged.
 * The `+=` augmented assignment operator is available in addition to `=` for
   normal assignment. Other augmented assignment operations aren't available.
 * Function arguments are evaluated at call time rather than at parse time; hence
   it is safe to use mutable defaults such as a list or dict.
 * Type annotations are available and are checked at runtime. Options can be combined
   with the `|` operator (e.g. `foo:list|dict=[]`).
 * `append` and `extend` on list objects are troublesome - our lists are a little
   more immutable (80% of the time they're immutable every time) than Python lists
   and hence those functions sort of work but don't have exactly the same semantics
   as the original (e.g. they may not always modify the original if it originated
   from a surrounding scope).
   Users are encouraged to use the `+=` augmented assignment operator instead.

The name is reminiscent of AST, and a play on an asp being a smaller snake than a python.
