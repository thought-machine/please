// +build windows

package core

import "os"

func acquireLockfile(lockFile *os.File){
  // TODO(jpoole): figure out how to Flock on windows
}

func releaseLockFile(lockFile *os.File) error {
	return nil
}