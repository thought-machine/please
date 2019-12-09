package lib

// vars that will be overridden at build time with actual git data.
// N.B. Must be a variable not a constant - constants aren't linker symbols and
//      hence can't be replaced in the same way.
var (
	GitRevision = "12345-revision"
	GitDescribe = "12345-describe"
)
