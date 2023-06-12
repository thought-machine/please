package debug

import (
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/exec"
	"github.com/thought-machine/please/src/process"
)

var log = logging.Log

func Debug(state *core.BuildState, label core.BuildLabel, args, env []string) int {
	target := state.Graph.TargetOrDie(label)
	if len(target.Debug.Command) == 0 {
		log.Fatalf("The build definition used by %s doesn't appear to support debugging yet", target.Label)
	}

	// Runtime directory.
	dir := filepath.Join(core.OutDir, "debug", target.Label.Subrepo, target.Label.PackageName)

	sandbox := target.Sandbox
	if target.IsTest() {
		sandbox = target.Test.Sandbox
		env = append(env, "TESTS="+strings.Join(args, " "))
	}
	// Append passed in arguments to the debug command.
	cmd := append(strings.Split(target.Debug.Command, " "), args...)

	// The value of `port` takes priority in deciding whether the network namespace should
	// be shared or not, otherwise clients (i.e. IDEs) might not be able to connect to the debugger.
	shareNetwork := state.DebugPort != 0 || !sandbox

	return exec.Exec(state, core.AnnotatedOutputLabel{BuildLabel: label}, dir, env, cmd, nil, state.DebugPort == 0, process.NewSandboxConfig(!shareNetwork, sandbox))
}
