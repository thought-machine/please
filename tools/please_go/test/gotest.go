package test

import "log"

func PleaseGoTest(dir, importPath, pkg, output string, sources, exclude []string, isBenchmark bool) {
	coverVars, err := FindCoverVars(dir, importPath, exclude, sources)
	if err != nil {
		log.Fatalf("Error scanning for coverage: %s", err)
	}
	if err = WriteTestMain(pkg, importPath, sources, output, dir != "", coverVars, isBenchmark); err != nil {
		log.Fatalf("Error writing test main: %s", err)
	}
}