ASP
===

This package implements a custom parser for BUILD files.

The so-far-unnamed BUILD language is (almost) a subset of Python,
with many of its more advanced or dynamic features stripped out. Obviously
it is not easy to implement even a subset of it and so many aspects
remain unimplemented. Some of the notable differences are:
 * The `import`, `try`, `except`, `finally`, `class`, `global`, `nonlocal`,
   `while` and `async` keywords are not available. It is therefore possible but
   discouraged to use these as identifiers.
 * The `raise` and `assert` statements are supported, but since it is not
   possible to catch exceptions they only serve to signal catastrophic errors.
 * List and dict comprehensions are supported, but not Python's more general
   generator expressions. Up to two 'for' clauses are permitted.
 * Most builtin functions are not available.
 * Dictionaries are supported, but can only be keyed by strings.
 * The only builtin types are `bool`, `int`, `str`, `list`, `dict` and functions.
   There are no `float`, `complex`, `set`, `frozenset` or `bytes` types.
 * Operators `+`, `<`, `>`, `%`, `and`, `or`, `in`, `not in`, `==`, `>=`,
   `<=` and `!=` are supported in most appropriate cases. Other operators
   are not available.
 * Limited string interpolation is available via `%`. `format()` is also available
   but its implementation is incomplete and use is discouraged.
 * The `+=` augmented assignment operator is available in addition to `=` for
   normal assignment. Other augmented assignment operations aren't available.
 * Operator precedence is not always the same as Python's at present.
   This may well change in later versions, although users are encouraged to
   avoid relying on precedence details where possible.
 * The accepted syntax likely differs in minor ways from Python's due to
   implementation details on our part or limitations in how our parser
   treats files. In general we try to match Python's semantics but only
   as a best-effort; many of the differences mentioned here are deliberate.
 * `append` and `extend` on list objects are troublesome - our lists are a little
   more immutable (80% of the time they're immutable every time) than Python lists
   and hence those functions sort of work but don't have exactly the same semantics
   as the original (e.g. they may not always modify the original if it originated
   from a surrounding scope).
   Users are encouraged to use the `+=` augmented assignment operator instead.
 * `subinclude()` has slightly different semantics now. Subincluded files must
   include their dependencies (it is not sufficient simply to do them in order).
   Subincludes should not attempt to mutate global objects.

TODOs (in no particular order of importance):
 * Better guarantees around ordering. Consider banishing sorted().
 * Better operator precedence.

The name is reminiscent of AST, and a play on an asp being a smaller snake than a python.
