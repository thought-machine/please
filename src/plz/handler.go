package plz

import (
	"core"
)

// Handle will handle commandline requests from the plz cli
func Handle(command string, initOpts InitOpts, params map[string]interface{}) bool {
	//argsIface, ok := params["Args"]
	//if !ok {
	//	return false
	//}
	//args, ok := argsIface.(map[string]interface{})
	//if !ok {
	//	return false
	//}
	//
	//targets := getTargets(args)
	funcs := map[string]func(initOpts *InitOpts, params map[string]interface{}) bool{
		"run":      handleRun,
		"build":    handleBuild,
		"rebuild":  handleRebuild,
		"hash":     handleHash,
		"test":     handleTest,
		"cover":    handleCover,
		"parallel": handleParallel,
	}

	return funcs[command](&initOpts, params)
}

func getTargets(args map[string]interface{}) []core.BuildLabel {
	if targetsIface, exist := args["Targets"]; exist {
		if targets, ok := targetsIface.([]core.BuildLabel); ok {
			return targets
		}
	}

	if targetIface, exist := args["Target"]; exist {
		if target, ok := targetIface.(core.BuildLabel); ok {
			return []core.BuildLabel{target}
		}
	}

	return nil
}
