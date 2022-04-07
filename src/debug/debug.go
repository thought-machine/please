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

func Debug(state *core.BuildState, label core.BuildLabel, args []string, workingDir string) int {
	target := state.Graph.TargetOrDie(label)
	if len(target.Debug.Command) == 0 {
		log.Fatalf("The build definition used by %s doesn't appear to support debugging yet", target.Label)
	}

	// Runtime directory.
	runtimeDir := filepath.Join(core.OutDir, "debug", target.Label.Subrepo, target.Label.PackageName)

	// Default non-test execution configuration.
	env := []string{}
	sandboxConfig := process.NoSandbox

	// Mimic test execution configuration.
	if target.IsTest() {
		env = append(env, "TESTS="+strings.Join(args, " "))
		// The value of `port` takes priority in deciding whether the network namespace should
		// be shared or not, otherwise clients (i.e. IDEs) might not be able to connect to the debugger.
		shareNetwork := state.DebugPort != 0 || !target.Test.Sandbox
		sandboxConfig = process.NewSandboxConfig(!shareNetwork, target.Test.Sandbox)
	}

	// Append passed in arguments to the debug command.
	cmd := append(strings.Split(target.Debug.Command, " "), args...)

	return exec.Exec(state, core.AnnotatedOutputLabel{BuildLabel: label}, runtimeDir, workingDir, env, cmd, state.DebugPort == 0, sandboxConfig)
}
