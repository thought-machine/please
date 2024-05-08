# Package remote

The remote package implements a client that can be used to build please targets with the 
[remote execution API](https://github.com/bazelbuild/remote-apis). To understand this, it's highly recommended to read 
though the documentation on the 
[remote execution proto](https://github.com/bazelbuild/remote-apis/blob/main/build/bazel/remote/execution/v2/remote_execution.proto). 

With that in mind, building with remote execution works as follows: 

1. Build the command and action protos from the Target. 
2. Check to see if we have a cached action result on the target metadata file
3. Check to see if we have a cached action result in the remote action cache. 
4. Otherwise, prepare the input directory, uploading any missing files required to build the action e.g. the srcs. Deps
   should already be in the CAS from when the dependency was built. 
5. Submit the action to be built, and save the action result on the target metadata file. Store this in the local cache. 

There are some prickly edge cases around filegroups and remote files. Filegroups don't have an action per-se, however we
generate a psudo-action and action result locally that is uploaded to the action cache. This will likely change in the 
future, as this isn't strictly necessary and confuses things. Remote files are built using the 
[remote asset API](https://github.com/bazelbuild/remote-apis/blob/main/build/bazel/remote/asset/v1/remote_asset.proto).