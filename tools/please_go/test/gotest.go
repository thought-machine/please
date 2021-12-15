package test

import (
	"log"

	"github.com/thought-machine/please/tools/please_go/install/toolchain"
)

// PleaseGoTest will generate the test main for the provided sources
func PleaseGoTest(goTool, dir, importPath, testPackage, output string, sources, exclude []string, isBenchmark, external bool) {
	coverVars, err := FindCoverVars(dir, importPath, testPackage, external, exclude, sources)
	if err != nil {
		log.Fatalf("Error scanning for coverage: %s", err)
	}
	tc := toolchain.Toolchain{GoTool: goTool}
	minor, err := tc.GoMinorVersion()
	if err != nil {
		log.Fatalf("Failed to determine Go version: %s", err)
	}
	if err := WriteTestMain(testPackage, sources, output, dir != "", coverVars, isBenchmark, minor >= 18); err != nil {
		log.Fatalf("Error writing test main: %s", err)
	}
}
