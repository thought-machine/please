package test

import "log"

// PleaseGoTest will generate the test main for the provided sources
func PleaseGoTest(dir, importPath, testPackage, output string, sources, exclude []string, isBenchmark, external bool) {
	coverVars, err := FindCoverVars(dir, importPath, testPackage, external, exclude, sources)
	if err != nil {
		log.Fatalf("Error scanning for coverage: %s", err)
	}
	if err = WriteTestMain(testPackage, sources, output, dir != "", coverVars, isBenchmark); err != nil {
		log.Fatalf("Error writing test main: %s", err)
	}
}
