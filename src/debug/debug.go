package debug

import (
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/exec"
	"github.com/thought-machine/please/src/process"
)

func Debug(state *core.BuildState, label core.BuildLabel) int {
	target := state.Graph.TargetOrDie(label)
	dir := filepath.Join(core.OutDir, "debug", target.Label.Subrepo, target.Label.PackageName)
	return exec.Exec(state, label, dir, strings.Split(target.Debug.Command, " "), true, process.NewSandboxConfig(target.Sandbox, target.Sandbox))
}
