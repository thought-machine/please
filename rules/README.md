Built-in build rules
--------------------

This directory contains the various built-in build rules for Please.
They are mostly split up by language; `builtins.build_defs` contains the
lowest-level builtin functions, many of which are known to the interpreter
and have custom implementations.

All of them are written in [the BUILD language](https://please.build/language.html).
