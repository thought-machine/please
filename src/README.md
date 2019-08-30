This directory holds the source code for Please itself (as opposed
to the various tools it invokes which are in the `tools` directory,
or built-in rules or tests etc).

A quick overview of the structure here:
 - build: Logic for actually building targets & managing incrementality
 - cache: The various cache implementations
 - clean: Logic for `plz clean`
 - cli: Support package for flags, logging, etc.
 - core: Central package with core data structures
 - export: Implementation of `plz export`
 - follow: Implementation of `plz follow`
 - fs: Filesystem operations
 - gc: Implementation of `plz gc` (garbage collection)
 - hashes: Implementation of `plz hash` (mostly hash updating)
 - ide: Implementation of `plz ide`
 - output: Logic for printing to terminal & showing interactive output
 - parse: Logic for parsing BUILD files
 - parse/asp: Lower-level parser implementation
 - plz: High-level logic to orchestrate a build
 - process: Subprocess control & monitoring
 - query: Implementation of the various `plz query` subcommands
 - remote: Higher-level interface to the remote execution API
 - run: Implementation of `plz run`
 - scm: Code for talking to the source control system (i.e. git)
 - test: Logic for testing targets, reading results & coverage
 - tool: Implementation of `plz tool`
 - update: Self-updating logic
 - utils: Utilities & poor code organisation :)
 - watch: Implementation of `plz watch`
 - worker: Code for handling background worker processes
