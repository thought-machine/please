package core

import (
	"os"
	"path/filepath"
)

// machineConfigFileName lives under ProgramData, which is the Windows equivalent of /etc for
// machine-wide configuration. If the variable isn't set we fall back to the conventional path.
var machineConfigFileName = filepath.Join(programData(), "please", "plzconfig")

// defaultPath is deliberately empty. There is no Windows equivalent of /usr/bin holding the
// tools a build might need, and the conventional locations (System32 and friends) hold none
// of them, so there is nothing useful to default to; users configure [build] path instead.
var defaultPath []string

func programData() string {
	if dir := os.Getenv("ProgramData"); dir != "" {
		return dir
	}
	return `C:\ProgramData`
}
