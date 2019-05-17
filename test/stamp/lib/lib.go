package lib

// GitRevision will be overridden at build time with the actual git revision.
// N.B. Must be a variable not a constant - constants aren't linker symbols and
//      hence can't be replaced in the same way.
var GitRevision = "12345"
