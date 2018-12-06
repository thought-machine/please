package plz

import (
	"core"
)

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
	//targets := GetTargets(args)
	funcs := map[string]func(initOpts *InitOpts, params map[string]interface{}) bool{
		"run":   handleRun,
		"build": handleBuild,
		"test":  handleTest,
		"cover": handleCover,
	}

	return funcs[command](&initOpts, params)
}

func GetTargets(args map[string]interface{}) []core.BuildLabel {
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
