package fs

import (
	"os"
	"runtime"
)

// We query the working directory at init, to use it later to search for the
// executable file
// errWd will be checked later, if we need to use initWd
var initWd, errWd = os.Getwd()

// Executable does exactly what os.Exceutable does however uses the path approach for FreeBSD instead of basing it off
// SYSCTL sysc call information, which seems to follow hardlinks somehow.
func Executable() (string, error) {
	if runtime.GOOS == "freebsd" {
		return executable()
	}
	return os.Executable()
}

// This is from from the os package
func executable() (string, error) {
	var exePath string
	if len(os.Args) == 0 || os.Args[0] == "" {
		return "", os.ErrNotExist
	}
	if os.IsPathSeparator(os.Args[0][0]) {
		// Args[0] is an absolute path, so it is the executable.
		// Note that we only need to worry about Unix paths here.
		exePath = os.Args[0]
	} else {
		for i := 1; i < len(os.Args[0]); i++ {
			if os.IsPathSeparator(os.Args[0][i]) {
				// Args[0] is a relative path: prepend the
				// initial working directory.
				if errWd != nil {
					return "", errWd
				}
				exePath = initWd + string(os.PathSeparator) + os.Args[0]
				break
			}
		}
	}
	if exePath != "" {
		if err := isExecutable(exePath); err != nil {
			return "", err
		}
		return exePath, nil
	}
	// Search for executable in $PATH.
	for _, dir := range splitPathList(os.Getenv("PATH")) {
		if len(dir) == 0 {
			dir = "."
		}
		if !os.IsPathSeparator(dir[0]) {
			if errWd != nil {
				return "", errWd
			}
			dir = initWd + string(os.PathSeparator) + dir
		}
		exePath = dir + string(os.PathSeparator) + os.Args[0]
		switch isExecutable(exePath) {
		case nil:
			return exePath, nil
		case os.ErrPermission:
			return "", os.ErrPermission
		}
	}
	return "", os.ErrNotExist
}

// isExecutable returns an error if a given file is not an executable.
func isExecutable(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := stat.Mode()
	if !mode.IsRegular() {
		return os.ErrPermission
	}
	if (mode & 0111) == 0 {
		return os.ErrPermission
	}
	return nil
}

// splitPathList splits a path list.
// This is based on genSplit from strings/strings.go
func splitPathList(pathList string) []string {
	if pathList == "" {
		return nil
	}
	n := 1
	for i := 0; i < len(pathList); i++ {
		if pathList[i] == os.PathListSeparator {
			n++
		}
	}
	start := 0
	a := make([]string, n)
	na := 0
	for i := 0; i+1 <= len(pathList) && na+1 < n; i++ {
		if pathList[i] == os.PathListSeparator {
			a[na] = pathList[start:i]
			na++
			start = i + 1
		}
	}
	a[na] = pathList[start:]
	return a[:na+1]
}
