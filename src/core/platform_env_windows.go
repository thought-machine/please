package core

// setPlatformTmpEnv sets any platform-specific environment variables pointing at a build
// action's temporary directory. Windows-native tools look at USERPROFILE rather than HOME,
// and at TEMP/TMP rather than TMPDIR, so they need the same redirection for the build
// environment to stay hermetic.
func setPlatformTmpEnv(env BuildEnv, dir string) {
	env["USERPROFILE"] = dir
	env["TEMP"] = dir
	env["TMP"] = dir
}
