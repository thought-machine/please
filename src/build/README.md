# Package build

This package defines the build step used to actually build a target. Targets are either built locally within a temp 
directory under `plz-out/tmp/...`, or remotely via the [remote execution API](https://github.com/bazelbuild/remote-apis).

Both local and remote execution pass through this step, however the codepaths are quite separate. For remote execution,
this package mainly passes off to `src/remote`.

There are a few concepts to understand:
1. Targets can be built and the assets in `plz-out` are present and up to date
2. Targets can be cached but the assets in `plz-out` are out of date or missing. We must restore the target from the 
   cache in this case.
3. The target might have changed and need to actually be built.
4. Targets have a metadata file that contains extra build information including the standard output of the build action
   required to run the post-build function. 

With that in mind, this package loosely follows these steps for local execution:
1. Check if we have already built the rule i.e. it's present in plz-out
   1. checks the hashes on xargs of all input files (rule, config, source, secret)
   2. re-applies any updates that might have happened during build (the post-build and output dirs)
   3. re-checks the hashes to see if those updates changed anything and need to re-build otherwise returns (nothing to do)
2. Checks if we have the target in the build cache
   1. if the action of building this target could've changed how we calculate the output hash (e.g. has a post-build),
      1. attempt to fetch just the metadata file from the cache based on the old hashkey
      2. apply these updates to the outs based on the stored metadata (out dirs + run post build action)
   2. attempt to fetch the outputs from the cache based on the output hash
3. Otherwise, actually build the rule
4. Store result in the cache