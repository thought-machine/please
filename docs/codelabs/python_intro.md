summary: Getting started with Python
description: Building and testing with Python and Please, as well as managing third-party dependencies via pip
id: python_intro
categories: beginner
tags: medium
status: Published
authors: Jon Poole
Feedback Link: https://github.com/thought-machine/please

# Getting started with Python
## Overview
Duration: 4

### Prerequisites
- You must have Please installed: [Install please](https://please.build/quickstart.html)
- Python must be installed: [Install Python](https://www.python.org/downloads/)

### What You’ll Learn
- Configuring Please for Python
- Creating an executable Python binary
- Authoring Python modules in your project
- Testing your code
- Including third-party libraries

### what if I get stuck?

The final result of running through this codelab can be found
[here](https://github.com/thought-machine/please-codelabs/tree/main/getting_started_python) for reference. If you really
get stuck you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## Initialising your project
Duration: 2

Let's create a new project:
```
$ mkdir getting_started_python && cd getting_started_python
$ plz init --no_prompt
```

### A note about your Please PATH
Please doesn't use your host system's `PATH` variable. By default, Please uses `/usr/local/bin:/usr/bin:/bin`. If Python
isn't in this path, you will need to add the following to `.plzconfig`:
```
[build]
path = $YOUR_PYTHON_INSTALL_HERE:/usr/local/bin:/usr/bin:/bin
```

### So what just happened?
You will see this has created a number of files in your working folder:
```
$ tree -a
  .
  ├── pleasew
  └── .plzconfig
```

The `pleasew` script is a wrapper script that will automatically install Please if it's not already! This
means Please projects are portable and can always be built via
`git clone https://... example_module && cd example_module && ./pleasew build`.

Finally, `.plzconfig` contains the project configuration for Please; read the [config](/config.html) documentation for
more information on configuration.

## Hello, world!
Duration: 3

Now we have a Please project, it's time to start adding some code to it! Let's create a "hello world" program:

### `src/main.py`
```python
print('Hello, world!')
```

We now need to tell Please about our Python code. Please projects define metadata about the targets that are available to be
built in `BUILD` files. Let's create a `BUILD` file to build this program:

### `src/BUILD`
```python
python_binary(
  name = "main",
  main = "main.py",
)
```

That's it! You can now run this with:
```
$ plz run //src:main
Hello, world!
```

There's a lot going on here; first off, `python_binary()` is one of many [built-in functions](/lexicon.html#python).
This build function creates a "build target" in the `src` package. A package, in the Please sense, is any directory that
contains a `BUILD` file.

Each build target can be identified by a build label in the format `//path/to/package:label`, i.e. `//src:main`.
There are a number of things you can do with a build target such e.g. `plz build //src:main`, however, as you've seen,
if the target is a binary, you may run it with `plz run`.

## Adding modules
Duration: 4

Let's add a `src/greetings` package to our Python project:

### `src/greetings/greetings.py`
```python
import random

def greeting():
    return random.choice(["Hello", "Bonjour", "Marhabaan"])
```

We then need to tell Please how to compile this library:

### `src/greetings/BUILD`
```python
python_library(
    name = "greetings",
    srcs = ["greetings.py"],
    visibility = ["//src/..."],
)
```
NB: Unlike many popular build systems, Please doesn't just have one metadata file in the root of the project. Please will
typically have one `BUILD` file per [Python package](https://docs.python.org/3/tutorial/modules.html#packages).

We can then build it like so:

```
$ plz build //src/greetings
Build finished; total time 290ms, incrementality 50.0%. Outputs:
//src/greetings:greetings:
  plz-out/gen/src/greetings/greetings.py
```

Here we can see that the output of a `python_library` rule is a `.py` file which is stored in
`plz-out/gen/src/greetings/greetings.py`.

We have also provided a `visibility` list to this rule. This is used to control where this `python_library()` rule can be
used within our project. In this case, any rule under `src`, denoted by the `...` syntax.

NB: This syntax can also be used on the command line, e.g. `plz build //src/...`.

### A note about `python_binary()`
If you're used to Python, one thing that might trip you up is how we package Python. The `python_binary()` rule outputs
something called a `pex`. This is very similar to the concept of a `.jar` file from the java world. All the Python files
relating to that build target are zipped up into a self-executable `.pex` file. This makes deploying and distributing
Python simple as there's only one file to distribute.

Check it out:
```
$ plz build //src:main
Build finished; total time 50ms, incrementality 100.0%. Outputs:
//src:main:
  plz-out/bin/src/main.pex

$ plz-out/bin/src/main.pex
Bonjour, world!
```

## Using our new module
Duration: 2

To maintain a principled model for incremental and hermetic builds, Please requires that rules are explicit about their
inputs and outputs. To use this new package in our "hello world" program, we have to add it as a dependency:

### `src/BUILD`
```python
python_binary(
    name = "main",
    main = "main.py",
    # NB: if the package and rule name are the same, you may omit the name i.e. this could be just //src/greetings
    deps = ["//src/greetings:greetings"],
)
```

You can see we use a build label to refer to another rule here. Please will make sure that this rule is built before
making its outputs available to our rule here.

Then update src/main.py:
### `src/main.py`
```python
from src.greetings import greetings

print(greetings.greeting() + ", world!")
```

Give it a whirl:

```
$ plz run //src:main
Bonjour, world!
```

## Testing our code
Duration: 5

Let's create a very simple test for our library:
### `src/greetings/greetings_test.py`
```python
import unittest
from src.greetings import greetings

class GreetingTest(unittest.TestCase):

    def test_greeting(self):
        self.assertTrue(greetings.greeting())

```

We then need to tell Please about our tests:
### `src/greetings/BUILD`
```python
python_library(
    name = "greetings",
    srcs = ["greetings.py"],
    visibility = ["//src/..."],
)

python_test(
    name = "greetings_test",
    srcs = ["greetings_test.py"],
    # Here we have used the shorthand `:greetings` label format. This format can be used to refer to a rule in the same
    # package and is shorthand for `//src/greetings:greetings`.
    deps = [":greetings"],
)
```

We've used `python_test()` to define our test target. This is a special build rule that is considered a test. These
rules can be executed as such:
```
$ plz test //src/...
//src/greetings:greetings_test 1 test run in 3ms; 1 passed
1 test target and 1 test run in 3ms; 1 passed. Total time 90ms.
```

Please will run all the tests it finds under `//src/...`, and aggregate the results up. This works even across
languages allowing you to test your whole project with a single command.

## Third-party dependencies
Duration: 7

### Using `pip_library()`

Eventually, most projects need to depend on third-party code. Let's include NumPy into our package. Conventionally,
third-party dependencies live under `//third_party/...` (although they don't have to), so let's create that package:

### `third_party/python/BUILD`
```python
package(default_visibility = ["PUBLIC"])

pip_library(
    name = "numpy",
    version = "1.18.4",
    zip_safe = False, # This is because NumPy has shared object files which can't be linked to them when zipped up
)
```

This will download NumPy for us to use in our project. We use the `package()` built-in function to set the default
visibility for this package. This can be very useful for third-party rules to avoid having to specify
`visibility = ["PUBLIC"]` on every `pip_library()` invocation.

NB: The visibility "PUBLIC" is a special case. Typically, items in the visibility list are labels. "PUBLIC" is equivalent
to `//...`.

### Setting up our module path
Importing Python modules is based on the import path. That means by default, we'd import NumPy as
`import third_party.python.numpy`. To fix this, we need to tell Please where our third-party module is. Add the
following to your `.plzconfig`:

### `.plzconfig`
```
[python]
moduledir = third_party.python
```

### Updating our tests

We can now use this library in our code:

### `src/greetings/greetings.py`
```go
from numpy import random

def greeting():
    return random.choice(["Hello", "Bonjour", "Marhabaan"])

```

And add NumPy as a dependency:
### `src/greetings/BUILD`
```python
python_library(
    name = "greetings",
    srcs = ["greetings.py"],
    visibility = ["//src/..."],
    deps = ["//third_party/python:numpy"],
)

python_test(
    name = "greetings_test",
    srcs = ["greetings_test.py"],
    deps = [":greetings"],
)
```

## What next?
Duration: 1

Hopefully you now have an idea as to how to build Python with Please. Please is capable of so much more though!

- [Please basics](/basics.html) - A more general introduction to Please. It covers a lot of what we have in this
tutorial in more detail.
- [Built-in rules](/lexicon.html#python) - See the rest of the Python rules as well as rules for other languages and tools.
- [Config](/config.html#python) - See the available config options for Please, especially those relating to Python.
- [Command line interface](/commands.html) - Please has a powerful command line interface. Interrogate the build graph,
determine files changes since master, watch rules and build them automatically as things change and much more! Use
`plz help`, and explore this rich set of commands!

Otherwise, why not try one of the other codelabs!
