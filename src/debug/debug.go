package debug

import (
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/exec"
	"github.com/thought-machine/please/src/process"
)

func Debug(state *core.BuildState, label core.BuildLabel, args []string) int {
	target := state.Graph.TargetOrDie(label)

	dir := filepath.Join(core.OutDir, "debug", target.Label.Subrepo, target.Label.PackageName)

	var env []string
	if target.IsTest() {
		env = []string{"TESTS=" + strings.Join(args, " ")}
	}
	cmd := append(strings.Split(target.Debug.Command, " "), args...)

	return exec.Exec(state, label, dir, env, cmd, true, process.NewSandboxConfig(target.Sandbox, target.Sandbox))
}
