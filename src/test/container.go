// Support for containerising tests. Currently Docker only.

package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"build"
	"core"
)

func runContainerisedTest(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	testDir := path.Join(core.RepoRoot, target.TestDir())
	replacedCmd := build.ReplaceTestSequences(target, target.GetTestCommand())
	replacedCmd += " " + strings.Join(state.TestArgs, " ")
	containerName := state.Config.Docker.DefaultImage
	if target.ContainerSettings != nil && target.ContainerSettings.DockerImage != "" {
		containerName = target.ContainerSettings.DockerImage
	}
	// Gentle hack: remove the absolute path from the command
	replacedCmd = strings.Replace(replacedCmd, testDir, "/tmp/test", -1)
	// Fiddly hack follows to handle docker run --rm failing saying "Cannot destroy container..."
	// "Driver aufs failed to remove root filesystem... device or resource busy"
	cidfile := path.Join(testDir, ".container_id")
	// Using C.UTF-8 for LC_ALL because it works. Not sure it's strictly
	// correct to mix that with LANG=en_GB.UTF-8
	command := []string{"docker", "run", "--cidfile", cidfile, "-e", "LC_ALL=C.UTF-8"}
	if target.ContainerSettings != nil {
		if target.ContainerSettings.DockerRunArgs != "" {
			command = append(command, strings.Split(target.ContainerSettings.DockerRunArgs, " ")...)
		}
		if target.ContainerSettings.DockerUser != "" {
			command = append(command, "-u", target.ContainerSettings.DockerUser)
		}
	} else {
		command = append(command, state.Config.Docker.RunArgs...)
	}
	for _, env := range core.BuildEnvironment(state, target, true) {
		command = append(command, "-e", strings.Replace(env, testDir, "/tmp/test", -1))
	}
	replacedCmd = "mkdir -p /tmp/test && cp -r /tmp/test_in/* /tmp/test && cd /tmp/test && " + replacedCmd
	command = append(command, "-v", testDir+":/tmp/test_in", "-w", "/tmp/test_in", containerName, "bash", "-o", "pipefail", "-c", replacedCmd)
	log.Debug("Running containerised test %s: %s", target.Label, strings.Join(command, " "))
	_, out, err := core.ExecWithTimeout(target, target.TestDir(), nil, target.TestTimeout, state.Config.Test.Timeout, state.ShowAllOutput, command)
	retrieveResultsAndRemoveContainer(target, cidfile, err == context.DeadlineExceeded)
	return out, err
}

func runPossiblyContainerisedTest(state *core.BuildState, target *core.BuildTarget) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	if target.Containerise {
		if state.Config.Test.DefaultContainer == core.ContainerImplementationNone {
			log.Warning("Target %s specifies that it should be tested in a container, but test "+
				"containers are disabled in your .plzconfig.", target.Label)
			return runTest(state, target)
		}
		out, err = runContainerisedTest(state, target)
		if err != nil && state.Config.Docker.AllowLocalFallback {
			log.Warning("Failed to run %s containerised: %s %s. Falling back to local version.",
				target.Label, out, err)
			return runTest(state, target)
		}
		return out, err
	}
	return runTest(state, target)
}

// retrieveResultsAndRemoveContainer copies the test.results file out of the Docker container and into
// the expected location. It then removes the container.
func retrieveResultsAndRemoveContainer(target *core.BuildTarget, containerFile string, warn bool) {
	cid, err := ioutil.ReadFile(containerFile)
	if err != nil {
		log.Warning("Failed to read Docker container file %s", containerFile)
		return
	}
	if !target.NoTestOutput {
		retrieveFile(target, cid, "test.results", warn)
	}
	if core.State.NeedCoverage {
		retrieveFile(target, cid, "test.coverage", false)
	}
	for _, output := range target.TestOutputs {
		retrieveFile(target, cid, output, false)
	}
	// Give this some time to complete. Processes inside the container might not be ready
	// to shut down immediately.
	timeout := core.State.Config.Docker.RemoveTimeout
	for i := 0; i < 5; i++ {
		cmd := []string{"docker", "rm", "-f", string(cid)}
		if _, err := core.ExecWithTimeoutSimple(timeout, cmd...); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// retrieveFile retrieves a single file (or directory) from a Docker container.
func retrieveFile(target *core.BuildTarget, cid []byte, filename string, warn bool) {
	log.Debug("Attempting to retrieve file %s for %s...", filename, target.Label)
	timeout := core.State.Config.Docker.ResultsTimeout
	cmd := []string{"docker", "cp", string(cid) + ":/tmp/test/" + filename, target.TestDir()}
	if out, err := core.ExecWithTimeoutSimple(timeout, cmd...); err != nil {
		if warn {
			log.Warning("Failed to retrieve results for %s: %s [%s]", target.Label, err, out)
		} else {
			log.Debug("Failed to retrieve results for %s: %s [%s]", target.Label, err, out)
		}
	}
}
