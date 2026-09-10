package process

// DefaultShell is busybox, which Please bundles in its Windows release. Windows has no system
// shell that can run a build action, so depending on one being installed isn't an option.
const DefaultShell = "busybox"

// DefaultShellArgs selects busybox's bash applet. Note it does not include --noprofile and
// --norc: busybox rejects both outright, and it reads no profile or rc files in the first
// place, so there is nothing to suppress.
var DefaultShellArgs = []string{"bash"}
