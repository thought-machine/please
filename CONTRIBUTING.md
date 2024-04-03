# How to help out

Want to contribute to please? Great! We're a small team, so we'd very much appreciate your help. This short document 
should help you get started. We're a friendly bunch, so we'd love for you to reach out! This guidance isn't meant to be 
dogmatic, it simply aims to reduce friction when trying to engage with the please community. 

## Read the contributor docs

There are README files sprinkled throughout this repo that try to explain different parts: 

- [src/core](src/core/README.md) contains the definitions for the core of Please including the build graph, state, 
  targets and labels. It also contains all the logic for queuing up and adding targets to the graph. 
- [src/parse](src/parse/README.md) defines the parse step, that parses and interprets the BUILD files, adding
  their targets to the build graph. 
- [src/build](src/build/README.md) similarly defines the build step. 
- [src/test](src/test/README.md)  similarly defines the test step. 
- [src/remote](src/remote/README.md) defines a remote execution client for build and testing using the remote execution 
  API. 

## Check out our closed issues first

Before you go any further, it is worth searching through our issues. While we aim to make please as hassle free to use,
there are still some common pitfalls. You may well find the answer you seek in this issues list. Perhaps you'll even 
find others with a similar problem to back you up! 

## Suggesting a change

Please is a versatile build system with a wide remit. There are certainly things that can be improved, and we welcome 
suggestions. However, there are some basic steps can be taken that will drastically improve the chances of seeing your 
changes realised.  

### Raise an issue first

This allows us to start the conversation early and potentially saves you a lot of time and effort before you get to 
work. A lot of the time the enhancement might already exist, perhaps it doesn't belong in the core please repo, or 
perhaps there's a good reason we don't want to make that change. Have you checked the 
[pleasings](https://github.com/thought-machine/pleasings) repo? There's a lot of useful auxiliary stuff there. Either 
way, opening the dialogue with us sooner rather than later will drastically reduce friction when you eventually open a 
pull request. 

When raising your issue, make sure to give us enough context. The please paradigm is reasonably opinionated. It's not 
unlikely that what you are trying to achieve can be achieved in a more pleasing way. Let us know what your ultimate goal 
is so that we can better understand how to best achieve this goal. 

### Get on gitter

As you work on please, you might struggle to find your way around the codebase. If you get stuck, you can always jump on 
[gitter](https://gitter.im/please-build/Lobby). There's usually somebody online to help you. While it may be tempting to 
jump on gitter right away, we'd prefer to have the initial discussion on the issue where it's visible historically, so 
please start there. Feel free to post a link to your issue in gitter though!

### Get to work

Once you and the please community have a good idea as to what you're trying to achieve, you should get to work. When 
doing so, be mindful of those that review your code. Pull requests should be small and focused. Wide sweeping refactors 
should not be mixed with features and bug fixes. Doing so will only slow down reviewing your code and causes friction. 

Your code should also have tests that demonstrate how it functions. The PR should have a good explanation as to what 
has changed and guidance to the reviewer as to how to test it. In general just make it as easy for us to gain confidence 
in your change, so we can approve your pull request and make you a contributor to the please build system!

### Add yourself to the contributors list

You've worked hard, and you deserve recognition. If this is your first contribution, feel free to add yourself to 
`/docs/acknowledgements.html` and be displayed on the please [website](https://please.build/acknowledgements.html).
