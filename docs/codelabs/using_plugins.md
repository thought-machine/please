summary: Using plugins
description: How to use Please's language plugins
id: using_plugins
categories: beginner
tags: medium
status: Published
authors: Sam Westmoreland
Feedback Link: https://github.com/thought-machine/please

# Using Plugins

## Overview

Duration: 1

### Prerequisites

- You must have Please installed: [Install please](https://please.build/quickstart.html)
- You should have a basic understanding of how to use Please to build and test code

### What you'll learn

Language plugins were introduced with the release of Please v17. Each plugin
contains build definitions specific to a particular language. In this codelab
we'll cover

- Where to find plugins
- How to can install them in your project
- How to configure them to work for your repo

## Initialising your Please repo

Duration: 1

For this codelab we'll start with a clean repo. The first thing you'll need to
do is initialise it as a Please repo. We can do this with `plz init`. Now if we
check the directory we should have a config file, as well as the Please wrapper
script `pleasew`:

```bash
plz init
tree -a
```

The output should look like this:
```bash
.
├── pleasew
└── .plzconfig

1 directory, 2 files
```

## Where to find plugins

Duration: 1

For a comprehensive list of available plugins, visit
[https://github.com/please-build/please-rules](https://github.com/please-build/please-rules).
There you'll find plugins for language build rules, plugins for various
technologies, plugins for generating protos, and tools to help you maintain
your Please project.

### Can't find a plugin for your language?

The plugin ecosystem was designed with extensibility in mind, so if there is a
language that you'd like to build with Please and no plugin, consider writing
one! The existing plugins should serve as helpful templates, and if you get
stuck, feel free to reach out to the Please team on Github or the Please
community on [Gitter](https://gitter.im/please-build/Lobby). There will also be
a codelab coming soon that will cover the basics of writing a new plugin.

## How to install a plugin

Duration: 4

The easy way to install a plugin in your project is to use `plz init`. We'll
use the Go plugin in this example:

```bash
plz init plugin go
tree -a
```

The output should look like this:
```bash
.
├── pleasew
├── plugins
│   └── BUILD
├── .plzconfig
└── plz-out
    └── log
        └── build.log

4 directories, 4 files
```

### `.plzconfig`

```ini
[parse]
preloadsubincludes = ///go//build_defs:go

[Plugin "go"]
Target = //plugins:go
```

In the plzconfig, we can see that two things have been added for us. The first
is a preloaded subinclude. This will ensure that whichever package we're in in
our project, the rules defined in the plugin we just installed will be available.
This is completely optional. If the intention is to only use the rules in a few
places, it might make more sense to have an explicit subinclude in those
packages to avoid the plugin being a dependency of the entire repo.

The second thing is a section for configuration of our new plugin. `Target` is
a required field in this section. This tells Please where to look for the build
target that defines the plugin. There may be other required fields depending on
the particular plugin we've installed. More information about the various config
options is available from the plugin repository itself (e.g. [https://github.com/
please-build/go-rules](https://github.com/please-build/go-rules)), or via `plz
help [language]`.

### `plugins/BUILD`

```python
plugin_repo(
  name = "go",
  revision = "v1.17.2",
  plugin = "go-rules",
  owner = "please-build",
)
```

A new file has also been created for us called `plugins/BUILD`. This file should
contain a `plugin_repo()` target which will download our desired plugin for us.
The plugin is actually defined as a *subrepo* under the hood, which is why when
we want to depend on the build definitions in the plugin, we reference them with
a `///` (like in the preloaded subinclude in the `.plzconfig` file). The `//` is
then used to reference build targets within that subrepo, `//build_defs:go` for
example.

Note the revision field will be set to the most recent available version of the
plugin, but can be set to any version tag or commit hash that you require.

The use of `plz init plugin` is entirely optional. You might prefer
to manually add the `plugin_repo()` target somewhere else if putting it in
`plugins/` doesn't fit your needs. The only requirements for using a plugin are
that there is a `plugin_repo()` target *somewhere*, and that it is referenced
in the `Target` field of the plugin's config section in the `.plzconfig` file.

## What's next?

Duration: 0

You should now be set up with a language plugin. You now have access to all of
the build definitions provided by your chosen plugin.

Go ahead and install any other plugins that you Please, and get building!
