package plz

import (
	"build"
	"cli"
	"core"
	"hashes"
	"os"
	"output"
	"run"
	"test"
)

func handleBuild(initOpts *InitOpts, params map[string]interface{}) bool {
	success, _ := runBuild(initOpts)
	return success
}

func handleRebuild(initOpts *InitOpts, params map[string]interface{}) bool {
	return handleBuild(initOpts, params)
}

func handleHash(initOpts *InitOpts, params map[string]interface{}) bool {
	success, state := runBuild(initOpts)
	if params["Detailed"].(bool) {
		for _, target := range state.ExpandOriginalLabels() {
			build.PrintHashes(state, state.Graph.TargetOrDie(target))
		}
	}
	if params["Update"].(bool) {
		hashes.RewriteHashes(state, state.ExpandOriginalLabels())
	}

	return success
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
	runArgs := params["Args"].(map[string]interface{})
	if success, state := runBuild(initOpts); success {
		run.Run(state, runArgs["Target"].(core.BuildLabel),
			runArgs["Args"].([]string), params["Env"].(bool))
	}
	return false
}

func handleParallel(initOpts *InitOpts, params map[string]interface{}) bool {
	if success, state := runBuild(initOpts); success {
		if params["Watch"].(bool) {
			run.Parallel(state, state.ExpandOriginalLabels(), params["Args"].([]string),
				params["NumTasks"].(int), params["Quiet"].(bool), params["Env"].(bool))
		} else {
			os.Exit(run.Parallel(state, state.ExpandOriginalLabels(), params["Args"].([]string),
				params["NumTasks"].(int), params["Quiet"].(bool), params["Env"].(bool)))
		}
	}
	return false
}
