# Package core

*This readme is a stub. Please raise an issue on this repo if you would like something expanded on.*

Core is the main package that everything else in Please hangs off. There are a number of central structures that are key 
to understanding Please: 

- BuildTarget - A task for the build system to perform. These are created by calls to `build_rule()`, `filegroup()` and 
  `remote_file()`
- BuildLabel - The labels i.e. `//third_party/go:testify`, which are used throughout Please to refer to build targets. 
- BuildGraph - Contains all the build targets and subrepos that we have parsed so far. 
- Configuration - represents the combined please configuration from `.plzconfig` files. 
- BuildState - Contains the state for a build including the build graph, config, and progress. BuildState is cloned for
  subrepos, but some parts are shared e.g. the build graph. The state object contains information like whether we should
  queue targets for test, or 
- Subrepo - A subrepo that has been added to the build graph. These are created for both calls to `subrepo()`, and for 
  architectures e.g. `///linux_amd64//third_party/go:testify`, or when passed `--arch` on the command line. 

# Queuing packages and targets 

This is very complex, but it's important to understand the following concepts: 

1. BuildState contains information about what kind of action Please is performing. If `state.ParseOnly` is true, we only
   queue targets up for parse. If `state.NeedsTest` is true, then we also queue targets up to be tested, otherwise we 
   only build targets. 
2. QueueTarget uses this to decide if a target needs to be activated. An activated target will be queued up to be built,
   otherwise, the target will simply be parsed and added the the build graph. 
3. Once a target is built, if state.NeedsTest is true, the target is queued up again, this time to be tested. 
4. Targets progress through various states defined in the `BuildTargetState` enum. This is also used to synchronise 
   various steps in the target lifecycle e.g. activating a target for build. Targets are added to the graph as 
   inactive, and later activated if they're required to be built. This enum is ordered, so represents the lifecycle of 
   the target.
5. Targets must be resolved before they are built. When targets are parsed, we only have the `BuildLabel` for their 
   dependencies. Resolving a target involves queuing up their dependencies so these labels can be resolved to the actual
   `BuildTarget`s. 
6. Dependencies must also be resolved through the [require/provide](https://please.build/require_provide.html) mechanism. 

All targets are queued up through `QueueTarget()`, regardless of if they're queued test, parse, or normal builds. This
loosely follows the following steps:

1. Check to see if the target is resolved i.e. exists in the graph. If it doesn't queue up the target's package for 
   parse. This will:
   1. Parse the package, calling `build_rule()` etc. to add the targets to the graph
   2. Activate the target for us, if necessary. This will queue up the target again for build, moving on to step 3.
2. If the target exists, and we need to build, and the target is not already activated, activate the target:
   1. Change the target status to active. 
   2. Continue to step 3
3. Resolve the [require/provide](https://please.build/require_provide.html) logic i.e. make the queued target provide 
   for the dependent target.
4. Call `queueResolvedTarget()` to queue the target up to be built, tested or parsed. This does the following async, in 
   a go-routine:
   1. Queue each dependency up to be resolved, which in turn follows the same logic here.
   2. Wait for these dependencies to be resolved by calling `target.resolveDependency()`. This syncs on a channel to 
      for the dependency to be added to the graph. These dependencies are registered to the target.dependencies field.
   3. Once the target has been resolved, we call `queueResolvedTarget()`, recursively following this logic. 
   4. If we're queuing the target for build, we now sync on waiting for each of these targets to be built. 
   5. Now that all our dependencies are build, if we need to build, then add a pending build.
5. Once a target has been built, the build step will queue the target up for test for us, if needed. 