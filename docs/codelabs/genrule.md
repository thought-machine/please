summary: Writing custom build definitions
description: Start here to learn how to write custom build rules to automate nearly anything in your build
id: genrule
categories: intermediate
tags: medium
status: Published
authors: Jon Poole
Feedback Link: https://github.com/thought-machine/please

# Custom build rules with `genrule()`
## Overview
Duration: 1

### Prerequisites
- You must have Please installed: [Install Please](https://please.build/quickstart.html)
- You should be comfortable using the existing build rules.

### What you'll learn
We'll be working through a contrived example writing a build definition for
[wc](https://www.gnu.org/software/coreutils/manual/html_node/wc-invocation.html#wc-invocation) from core utils.
In doing so you'll:
- Be introduced to genrule(), the generic build rule
- Explore the build environment with `--shell`
- Write and use custom build rule definitions
- Manage and write custom tools for your build definition
- Add configuration for your build definitions

### What if I get stuck?

The final result of running through this codelab can be found
[here](https://github.com/thought-machine/please-codelabs/tree/main/custom_rules) for reference. If you really get stuck
you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## genrule()
Duration: 3

Before we jump into writing custom build definitions, let me introduce you to `genrule()`, the generic build rule. Let's
just create a new project and initialise Please in it:
```bash
mkdir custom_rules && cd custom_rules
plz init --no_prompt
```

Then create a `BUILD` file in the root of the repository like so:
### `BUILD`
```python
genrule(
    name = "word_count",
    srcs = ["file.txt"],
    deps = [],
    cmd = "wc $SRC > $OUT",
    outs = ["file.wc"],
)
```

Then create file.txt:
```bash
echo "the quick brown fox jumped over the lazy dog" > file.txt
```

and build it:

```bash
$ plz build //:word_count
Build finished; total time 70ms, incrementality 0.0%. Outputs:
//:word_count:
  plz-out/gen/file.wc

$ cat plz-out/gen/file.wc
 1  9 45 file.txt
```

### Troubleshooting: "can't store data at section "scm""

This message means the runner is using an older Please release that doesn’t understand the `[scm]` section in your `.plzconfig`, so parsing fails before any build work begins.

**How to fix**
- Upgrade the Please version invoked in CI (pin the same version locally via `pleasew`, `setup-please`, or `PLZ_VERSION`).
- If upgrading immediately is impractical, temporarily remove or comment the `[scm]` block until the runner is updated.

### So what's going on?
Here we've used one of the built-in rules, `genrule()`, to run a custom command. `genrule()` can take a number of
parameters, most notably: the name of the rule, the inputs (sources and dependencies), its outputs, and the command
we want to run. The full list of available arguments can be found on the [`genrule()`](/lexicon.html#genrule)
documentation.

Here we've used it to count the number of words in `file.txt`. Please has helpfully set up some environment variables
that help us find our inputs, as well as where to put our outputs:

- `$SRC` - Set when there's only one item in the `srcs` list. Contains the path to that source file.
- `$SRCS` - Contains a space-separated list of the sources of the rule.
- `$OUT` - Set when there's only one item in the `outs` list. Contains the expected path of that output.
- `$OUTS` - Contains a space-separated list of the expected paths of the outputs of the rule.

For a complete list of available variables, see the [build env](/build_rules.html#build-env) docs.

The command `wc $SRC > $OUT` is therefore translated into `wc file.txt > file.wc` and we can see that the output of the
rule has been saved to `plz-out/gen/file.wc`.

## The build directory
Duration: 7

One of the key features of Please is that builds are hermetic, that is, commands are executed in an isolated and
controlled environment. Rules can't access files or env vars that are not explicitly made available to them. As a
result, incremental builds very rarely break when using Please.

Considering this, debugging builds would be quite hard if we couldn't play around in this build environment. Luckily,
Please makes this trivial with the `--shell` flag:

```
$ plz build --shell :word_count
Temp directories prepared, total time 50ms:
  //:word_count: plz-out/tmp/word_count._build
    Command: wc $SRC > $OUT

bash-4.4$ pwd
<snip>/plz-out/tmp/word_count._build

bash-4.4$ wc $SRC > $OUT

bash-4.4$ cat $OUT
 1  9 45 file.txt
```

As we can see, Please has prepared a temporary directory for us under `plz-out/tmp`, and put us in a true-to-life bash
environment. You may run `printenv`, to see the environment variables that Please has made available to us:

```
bash-4.4$ printenv
OS=linux
ARCH=amd64
LANG=en_GB.UTF-8
TMP_DIR=<snip>/plz-out/tmp/word_count._build
CMD=wc $SRC > $OUT
OUT=<snip>/plz-out/tmp/word_count._build/file.wc
TOOLS=
SRCS=file.txt
PKG=
CONFIG=opt
PYTHONHASHSEED=42
SRC=file.txt
OUTS=file.wc
PWD=<snip>/plz-out/tmp/word_count._build
HOME=<snip>/plz-out/tmp/word_count._build
NAME=word_count
TMPDIR=<snip>/plz-out/tmp/word_count._build
BUILD_CONFIG=opt
XOS=linux
XARCH=x86_64
SHLVL=1
PATH=<snip>/.please:/usr/local/bin:/usr/bin:/bin
GOOS=linux
PKG_DIR=.
GOARCH=amd64
_=/usr/bin/printenv
```

As you can see, the rule doesn't have access to any of the variables from the host machine. Even `$PATH` has been set
based on configuration in `.plzconfig`:

The `--shell` flag works for all targets (except filegroups), which of course means any of the built-in rules! Note,
`--shell` also works on `plz test`. You can `plz build --shell //my:test` to see how the test is built, and then
`plz test --shell //my:test` to see how it will be run.

## Build definitions
Duration: 5

We've managed to write a custom rule to count the number of words in `file.txt`, however, we have no way of reusing this,
so let's create a `wordcount()` build definition!

A build definition is just a function that creates one or more build targets which define how to build something. These
are typically defined inside `.build_def` files within your repository. Let's just create a folder for our definition:

### `build_defs/word_count.build_defs`
```python
def word_count(name:str, file:str) -> str:
    return genrule(
        name = name,
        srcs = [file],
        outs = [f"{name}.wc"],
        cmd = "wc $SRC > $OUT",
    )
```

We then need some way to access these build definitions from other packages. To do this, we typically use a filegroup:

### `build_defs/BUILD`
```python
filegroup(
    name = "word_count",
    srcs = ["word_count.build_defs"],
    visibility = ["PUBLIC"],
)
```

We can then use this in place of our `genrule()`:

### `BUILD`
```python
subinclude("//build_defs:word_count")

word_count(
    name = "word_count",
    file = "file.txt",
)
```

And check it still works:

```bash
plz build //:word_count
```
The output:

```
Build finished; total time 30ms, incrementality 100.0%. Outputs:
//:word_count:
  plz-out/gen/word_count.wc
```

### `subinclude()`
Subinclude is primarily used for including build definitions into your `BUILD` file. It can be thought of like a
Python import except it operates on a build target instead. Under the hood, subinclude parses the output of the target
and makes the top-level declarations available in the current package's scope.

The build target is usually a filegroup, however, this doesn't have to be the case. In fact, the build target can be
anything that produces parsable outputs.

It's almost always a bad idea to build anything as part of a subinclude. These rules will be built at parse time,
which can be hard to debug, but more importantly, will block the parser while it waits for that rule to build. Use
non-filegroup subincludes under very careful consideration!

## Managing tools
Duration: 7

Right now we're relying on `wc` to be available on the configured path. This is a pretty safe bet, however, Please
provides a powerful mechanism for managing tools, so let's over-engineer this:

### `build_defs/word_count.build_defs`
```python
def word_count(name:str, file:str, wc_tool:str="wc") -> str:
    return genrule(
        name = name,
        srcs = [file],
        outs = [f"{name}.wc"],
        cmd = "$TOOLS_WC $SRC > $OUT",
        tools = {
            "WC": [wc_tool],
        }
    )
```

Here we've configured our build definition to take the word count tool in as a parameter. This is then passed to
`genrule()` via the `tools` parameter. Please has set up the `$TOOLS_WC` environment variable which we can used to
locate our tool. The name of this variable is based on the key in this dictionary.

In this contrived example, this may not seem very useful, however, Please will perform some important tasks for us:

- If the tool is a program, Please will check it's available on the path at parse time.
- If the tool is a build rule, Please will build this rule and configure `$TOOLS_WC` so it can be invoked. Whether the
tool is on the path or a build rule is transparent to you, the rule's author!

### Custom word count tool
Currently, our word count rule doesn't just get the word count: it also gets the character and line count as well. I
mentioned that these can be build rules so let's create a true word count tool that counts just words:

### `tools/wc.sh`
```shell script
#!/bin/bash

wc -w $@
```

### `tools/BUILD`
```python
filegroup(
    name = "wc",
    srcs = ["wc.sh"],
    binary = True,
    visibility = ["PUBLIC"],
)
```

and let's test that out:

```
$ plz run //tools:wc -- file.txt
9 file.txt
```

Brilliant! We can now use this in our build rule like so:

### `BUILD`
```python
subinclude("//build_defs:word_count")

word_count(
    name = "lines_words_and_chars",
    file = "file.txt",
)

word_count(
    name = "just_words",
    file = "file.txt",
    wc_tool = "//tools:wc",
)
```

and check it all works:

```
$ plz build //:lines_words_and_chars //:just_words
Build finished; total time 30ms, incrementality 100.0%. Outputs:
//:lines_words_and_chars:
  plz-out/gen/lines_words_and_chars.wc
//:just_words:
  plz-out/gen/just_words.wc

$ cat plz-out/gen/lines_words_and_chars.wc
1  9 45 file.txt

$ cat plz-out/gen/just_words.wc
9 file.txt
```

## Configuration
Duration: 6

Right now, we have to specify the new word count tool each time we use our build definition! Let's have a look at how we
can configure this in our `.plzconfig` instead:

### `.plzconfig`
```
[Buildconfig]
word-count-tool = //tools:wc
```

The `[buildconfig]` section can be used to add configuration specific to your project. By adding the `word-count-tool`
config option here, we can use this in our build definition:

### `build_defs/word_count.build_defs`
```python
def word_count(name:str, file:str, wc_tool:str=CONFIG.WORD_COUNT_TOOL) -> str:
    return genrule(
        name = name,
        srcs = [file],
        outs = [f"{name}.wc"],
        cmd = "$TOOLS_WC $SRC > $OUT",
        tools = {
            "WC": [wc_tool],
        }
    )

CONFIG.setdefault('WORD_COUNT_TOOL', 'wc')
```

Here we've set the default value for `wc_tool` to `CONFIG.WORD_COUNT_TOOL`, which will contain our config value from
`.plzconfig`. What if that's not set though? That's why we also set a sensible default configuration value with
`CONFIG.setdefault('WORD_COUNT_TOOL', 'wc')`!


We then need to update our build rules:

### `BUILD`
```python
subinclude("//build_defs:word_count")

word_count(
    name = "lines_words_and_chars",
    file = "file.txt",
    wc_tool = "wc",
)

word_count(
    name = "just_words",
    file = "file.txt",
)
```

and check it all works:

```
$ plz build //:lines_words_and_chars //:just_words
Build finished; total time 30ms, incrementality 100.0%. Outputs:
//:lines_words_and_chars:
  plz-out/gen/lines_words_and_chars.wc
//:just_words:
  plz-out/gen/just_words.wc

$ cat plz-out/gen/lines_words_and_chars.wc
1  9 45 file.txt

$ cat plz-out/gen/just_words.wc
9 file.txt
```

## Conclusion
Duration: 2

Congratulations! You've written your first build definition! While contrived, this example demonstrates most of the
mechanisms used to create a rich set of build definitions for a new language or technology. To get a better understanding
of build rules, I recommend reading through the advanced topics on [please.build](/build_rules.html).

If you create something you believe will be useful to the wider world, we might be able to find a home for it in the
[pleasings](https://github.com/thought-machine/pleasings) repo!

If you get stuck, jump on [gitter](https://gitter.im/please-build/Lobby) and we'll do our best to help you!

