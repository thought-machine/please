package plz

import (
	"cli"
	"core"
	"os"
	"test"
)

func runBuild(initOpts *InitOpts) (bool, *core.BuildState) {
	if len(initOpts.Targets) == 0 {
		initOpts.Targets = core.InitialPackage()
	}

	return Init(*initOpts)
}

func doTest(initOpts *InitOpts, surefireDir cli.Filepath, resultsFile cli.Filepath) (bool, *core.BuildState) {
	os.RemoveAll(string(surefireDir))
	os.RemoveAll(string(resultsFile))
	os.MkdirAll(string(surefireDir), core.DirPermissions)
	success, state := runBuild(initOpts)
	test.CopySurefireXmlFilesToDir(state, string(surefireDir))
	test.WriteResultsToFileOrDie(state.Graph, string(resultsFile))
	return success, state
}
