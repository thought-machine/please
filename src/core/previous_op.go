package core

import (
	"os"
	"strings"

	"github.com/pkg/xattr"

	"github.com/thought-machine/please/src/fs"
)

const previousOpFilePath = "plz-out/.previous_op"

// StoreCurrentOperation stores the current operation being performed in a storage file that can later be
// used by `plz op`. It does not error out if it can't replace the contents of the file with the operation.
func StoreCurrentOperation() {
	file, err := fs.OpenDirFile(previousOpFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	previousOp := strings.Join(os.Args[1:], " ")
	if err := file.Truncate(0); err != nil {
		log.Errorf("Unable to truncate %s to store current operation: %s", previousOpFilePath, err)
	} else if _, err := file.WriteAt([]byte(previousOp+"\n"), 0); err != nil {
		log.Errorf("Unable to store current operation to  %s: %s", previousOpFilePath, err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}

// ReadPreviousOperationOrDie reads the previous operation performed from storage file. Dies if unsuccessful.
func ReadPreviousOperationOrDie() []string {
	contents, err := os.ReadFile(previousOpFilePath)
	if err != nil || len(contents) == 0 {
		log.Fatalf("Sorry OP, can't read previous operation :(")
	}
	return strings.Split(strings.TrimSpace(string(contents)), " ")
}

// CheckXattrsSupported leverages the file used to store the current operation to check for xattrs.
func CheckXattrsSupported(state *BuildState) {
	if !state.XattrsSupported {
		return
	}

	file, err := fs.OpenDirFile(previousOpFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Quick test of xattrs; we don't keep trying to use them if they fail here.
	if err := xattr.Set(file.Name(), "user.plz_build", []byte("op")); err != nil {
		log.Warning("xattrs are not supported on this filesystem, using fallbacks")
		state.DisableXattrs()
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}
