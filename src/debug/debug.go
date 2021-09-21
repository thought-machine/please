package debug

import (
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/exec"
	"github.com/thought-machine/please/src/process"
)

func Debug(state *core.BuildState, label core.BuildLabel, port int, args []string) int {
	target := state.Graph.TargetOrDie(label)

	dir := filepath.Join(core.OutDir, "debug", target.Label.Subrepo, target.Label.PackageName)

	sandbox := target.Sandbox
	env := []string{}
	if target.IsTest() {
		sandbox = target.Test.Sandbox
		env = append(env, "TESTS="+strings.Join(args, " "))
	}
	cmd := append(strings.Split(target.Debug.Command, " "), args...)

	exposePort := port != 0
	return exec.Exec(state, label, dir, env, cmd, !exposePort, process.NewSandboxConfig(exposePort || sandbox, sandbox))
}
