package update

import "core"

// CheckAndUpdate is a stub implementation that does nothing.
func CheckAndUpdate(config *core.Configuration, updatesEnabled, updateCommand, forceUpdate, verify bool) {
}

// DownloadPyPy is also a stub that does nothing.
func DownloadPyPy(config *core.Configuration) bool {
	return false
}
