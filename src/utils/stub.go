// +build bootstrap

package utils

// InitConfig is a stub used during initial bootstrap.
func InitConfig(dir string, bazelCompatibility bool) {
	log.Fatalf("Not supported during initial bootstrap.\n")
}

// InitConfigFile is a stub used during initial bootstrap.
func InitConfigFile(filename string, options map[string]string) {
}

// PrintCompletionScript is a stub used during initial bootstrap.
func PrintCompletionScript() {
	log.Fatalf("Not supported during initial bootstrap.\n")
}
