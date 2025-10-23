summary: Running Please on GitHub Actions
description: GitHub Actions is an extensible CI/CD platform provided by GitHub
id: github_actions
categories: intermediate
tags: medium
status: Published
authors: Márk Sági-Kazár
Feedback Link: https://github.com/thought-machine/please

# Running Please on GitHub Actions
## Overview
Duration: 2

### Prerequisites
- A repository on [GitHub](https://github.com)
- A project with Please initialized in that repository

### What you'll learn
- How to setup [GitHub Actions](https://github.com/features/actions)
- How to use Please in a GitHub Actions build
- How to use the [setup-please](https://github.com/sagikazarmark/setup-please-action) action for better integration

### What if I get stuck?
If you get stuck with GitHub Actions, check out the [official documentation](https://docs.github.com/en/free-pro-team@latest/actions).

You can find usage examples of the [setup-please](https://github.com/sagikazarmark/setup-please-action) action in [this](https://github.com/sagikazarmark/todobackend-go-kit/blob/20292fc09e25196e751e087da7c5e659cd6c452f/.github/workflows/ci.yaml) repository.

If you really get
stuck you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## GitHub Actions
Duration: 5

GitHub Actions is GitHub's built-in automation platform for CI/CD and other workflows. It runs workflows defined as YAML files in the .github/workflows directory, triggered by events (push, pull_request, schedule, manual, etc.). Workflows consist of jobs (run on hosted or self‑hosted runners) and steps that execute shell commands or reusable actions from the marketplace. Key benefits include tight GitHub integration, flexible triggers and matrices, a large action marketplace, and caching for faster builds.

### Setting up GitHub Actions

Workflow definitions are simple YAML files stored in the `.github/workflows` directory of your repository.

The following snippet triggers a workflow named `CI` whenever commits are pushed to the `master` branch:

```yaml
name: CI

on:
  push:
    branches:
      - master
  pull_request:

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v2

      - name: Test
        run: echo "Tests passed"
```

Go ahead and add the above snippet to `.github/workflows/ci.yaml` in your project. Then go to `https://github.com/YOU/YOUR-PROJECT/actions` and observe the workflow.

## Please build
Duration: 4

Now we have a project setup with GitHub Actions, it's time to start building with Please! Let's change `ci.yaml` a little:

```yaml
name: CI

on:
  push:
    branches:
      - master
  pull_request:

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest

    steps:
      # Setup your language of choice here:
      # https://github.com/actions/?q=setup-&type=&language=

      - name: Checkout code
        uses: actions/checkout@v2

      # Run please build
      - name: Test
        run: ./pleasew build //...
```

Compared to the example earlier, this workflow uses the `pleasew` script to download Please and build the project.

Notice the `//...` bit at the end of the command: it's necessary on GitHub Actions.
Check [this](https://github.com/thought-machine/please/issues/1174) issue for more details.

## setup-please action
Duration: 10

The [setup-please](https://github.com/sagikazarmark/setup-please-action) action provides better integration for Please.

### What is an _action_?

As you've seen in the previous examples, workflows consist of _steps_.
A workflow step can be as simple as a shell script:

```yaml
- name: Test
  run: ./pleasew build //...
```

Shell scripts (no matter how awesome they are) are not always the right tool for the job. Complex build steps might require a more expressive language which takes us to the second type of workflow steps, called _actions_:

```yaml
- name: Checkout code
  uses: actions/checkout@v2
```

An _action_ can be written in any language (distributed as Docker images), but JavaScript is supported natively.

### Why not just use ./pleasew?

The above section about _actions_ begs the question: why not just use `pleasew`? Why do we need an action for running Please.

Please itself can perfectly run on GitHub Actions on its own, so you don't need an _action_ per se. That being said, there are a couple issues when using `pleasew`:

- The wrapper script does not understand Please configuration which can lead to multiple downloads of different versions to different locations which takes time and time is expensive in CI.
- When using self-hosted runners, GitHub Actions offers a cache specifically for tools (like Please) that can further speed up workflows, but it requires a custom action.

The [setup-please](https://github.com/sagikazarmark/setup-please-action) action provides better integration for Please solving the above issues (and a lot more).

### Using the setup-please action

Adding the [setup-please](https://github.com/sagikazarmark/setup-please-action) action to your workflow is simply adding two lines:

```yaml
name: CI

on:
  push:
    branches:
      - master
  pull_request:

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest

    steps:
      # Setup your language of choice here:
      # https://github.com/actions/?q=setup-&type=&language=

      - name: Checkout code
        uses: actions/checkout@v2

      # Make sure it's added after the checkout step
      - name: Set up Please
        uses: sagikazarmark/setup-please-action@v0

      # Run please build
      # You can use plz thanks to the setup action
      - name: Test
        run: plz test //...
```

The readme of [setup-please](https://github.com/sagikazarmark/setup-please-action) explains more use cases and configuration options:

- global include/exclude labels
- global profile
- saving logs as artifacts
