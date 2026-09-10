package process

// shellInitArgs is empty on Windows. The shell there is busybox, whose bash applet rejects
// --noprofile and --norc outright; it reads no profile or rc files in the first place, so
// there is nothing to suppress.
var shellInitArgs []string
