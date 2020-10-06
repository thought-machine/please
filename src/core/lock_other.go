// +build !windows

package core

func acquireLockfile(lockFile *os.File){
	// Try a non-blocking acquire first so we can warn the user if we're waiting.
	log.Debug("Attempting to acquire lock %s...", lockFilePath)
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		log.Debug("Acquired lock %s", lockFilePath)
	} else {
		log.Warning("Looks like another plz is already running in this repo. Waiting for it to finish...")
		if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
			log.Fatalf("Failed to acquire lock: %s", err)
		}
	}
}

func releaseLockFile(lockFile *os.File) error {
	return syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
}