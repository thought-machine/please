package core

// sandboxSupported reports whether this platform can isolate a build action at all.
//
// Nothing on Windows does yet. The pieces exist - job objects, restricted tokens - but there
// is no analogue of a mount namespace, so filesystem isolation would need Windows Containers,
// which is far too large a dependency to take on. See docs/design/windows.
func sandboxSupported() bool { return false }
