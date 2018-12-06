package plz

import (
	"cli"
	"os"
	"output"
	"test"
)

func handleBuild(initOpts *InitOpts, params map[string]interface{}) bool {
	success, _ := runBuild(initOpts)
	return success
}

func handleRebuild(initOpts *InitOpts, params map[string]interface{}) bool {
	return false
}

func handleTest(initOpts *InitOpts, params map[string]interface{}) bool {
	failingTestsOk := params["FailingTestsOk"].(bool)

	success, _ := doTest(initOpts, params["SurefireDir"].(cli.Filepath),
		params["TestResultsFile"].(cli.Filepath))

	return success || failingTestsOk
}

func handleCover(initOpts *InitOpts, params map[string]interface{}) bool {
	sureFireDir := params["SurefireDir"].(cli.Filepath)
	resultsFile := params["TestResultsFile"].(cli.Filepath)
	failingTestsOk := params["FailingTestsOk"].(bool)

	os.RemoveAll(string(params["CoverageResultsFile"].(cli.Filepath)))
	success, state := doTest(initOpts, sureFireDir, resultsFile)

	test.AddOriginalTargetsToCoverage(state, params["IncludeAllFiles"].(bool))
	test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension)

	test.WriteCoverageToFileOrDie(state.Coverage, string(params["CoverageResultsFile"].(cli.Filepath)))
	test.WriteXMLCoverageToFileOrDie(initOpts.Targets, state.Coverage, string(params["CoverageXMLReport"].(cli.Filepath)))

	if params["LineCoverageReport"].(bool) {
		output.PrintLineCoverageReport(state, params["IncludeFile"].([]string))
	} else if params["NoCoverageReport"].(bool) {
		output.PrintCoverage(state, params["IncludeFile"].([]string))
	}
	return success || failingTestsOk
}

func handleRun(initOpts *InitOpts, params map[string]interface{}) bool {
	return false
}
