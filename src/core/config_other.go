//go:build !windows
// +build !windows

package core

const machineConfigFileName = "/etc/please/plzconfig"

var defaultPath = []string{"/usr/local/bin", "/usr/bin", "/bin"}
