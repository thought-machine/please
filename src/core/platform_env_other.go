//go:build !windows
// +build !windows

package core

// setPlatformTmpEnv sets any platform-specific environment variables pointing at a build
// action's temporary directory. Unix tools use HOME and TMPDIR, which are set already.
func setPlatformTmpEnv(env BuildEnv, dir string) {}
