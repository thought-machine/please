package langserver

// BuiltInsWithIrregularArgs is the list of builtin functions which arguments
// not matching the declared arguments
// Used in diagnostics.go: diagnoseFuncCall
var BuiltInsWithIrregularArgs = []string{"format", "zip", "package", "join_path"}

// LocalSrcsArgs is the list of arguments within build rules
// that may yield to local source files
var LocalSrcsArgs = []string{"srcs", "src", "main", "data", "hdrs", "private_hdrs"}
