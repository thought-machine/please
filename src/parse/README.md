# Package parse

*This readme is a stub. Please raise an issue on this repo if you would like something expanded on.*

This package defines the parse step used to parse, and interpret a BUILD file to populate the build graph. You can find 
the actual interpreter for asp, the python dialect used in build files, in the asp subfolder. 

Parsing does the following basic things:

1. Synchronise on package parsing by calling `state.SyncParsePackage(label)`. This will either return the existing 
   package, blocking if the parse is in flight, or return nil, if we're the first. When this return nil, we MUST parse 
   the package. 
2. Check to see if we have a subrepo label. When we do, the subrepo package must be parsed first. This involves waiting
   for the subrepo target that defiens this package to be built. 
3. Parse the package, and add the package to the build graph. We also mark the pacakge as parsed which unblocks any other
   calls to `state.SyncParsePackage(label)`. 
4. If we queued up a specific target to be built, activate the target and queue it again. This will trigger a build.
   see [src/core](../core/README.md) for more information on how this works.
