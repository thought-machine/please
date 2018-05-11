//+build !bootstrap

// Support for containerising tests. Currently Docker only.

package test

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"docker.io/go-docker"
	"docker.io/go-docker/api"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/container"
	"docker.io/go-docker/api/types/mount"

	"build"
	"core"
)

var dockerClient *docker.Client
var dockerClientOnce sync.Once

func runContainerisedTest(tid int, state *core.BuildState, target *core.BuildTarget) (out []byte, err error) {
	const testDir = "/tmp/test"
	const resultsFile = testDir + "/test.results"

	dockerClientOnce.Do(func() {
		dockerClient, err = docker.NewClient(docker.DefaultDockerHost, api.DefaultVersion, nil, nil)
		if err != nil {
			log.Error("%s", err)
		} else {
			dockerClient.NegotiateAPIVersion(context.Background())
			log.Debug("Docker client negotiated API version %s", dockerClient.ClientVersion())
		}
	})
	if err != nil {
		return nil, err
	} else if dockerClient == nil {
		return nil, fmt.Errorf("failed to initialise Docker client")
	}

	targetTestDir := path.Join(core.RepoRoot, target.TestDir())
	replacedCmd := build.ReplaceTestSequences(state, target, target.GetTestCommand(state))
	replacedCmd += " " + strings.Join(state.TestArgs, " ")
	// Gentle hack: remove the absolute path from the command
	replacedCmd = strings.Replace(replacedCmd, targetTestDir, targetTestDir, -1)

	env := core.TestEnvironment(state, target, testDir)
	env.Replace("RESULTS_FILE", resultsFile)
	env.Replace("GTEST_OUTPUT", "xml:"+resultsFile)

	config := &container.Config{
		Image: state.Config.Docker.DefaultImage,
		// TODO(peterebden): Do we still need LC_ALL here? It was kinda hacky before...
		Env:        append(env, "LC_ALL=C.UTF-8"),
		WorkingDir: testDir,
		Cmd:        []string{"bash", "-uo", "pipefail", "-c", replacedCmd},
		Tty:        true, // This makes it a lot easier to read later on.
	}
	hostConfig := &container.HostConfig{}
	// Bind-mount individual files in (not a directory) to avoid ownership issues.
	for out := range core.IterRuntimeFiles(state.Graph, target, false) {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: path.Join(core.RepoRoot, out.Src),
			Target: path.Join(config.WorkingDir, out.Tmp),
		})
	}
	if target.ContainerSettings != nil {
		if target.ContainerSettings.DockerImage != "" {
			config.Image = target.ContainerSettings.DockerImage
		}
		config.User = target.ContainerSettings.DockerUser
		if target.ContainerSettings.Tmpfs != "" {
			hostConfig.Tmpfs = map[string]string{target.ContainerSettings.Tmpfs: "exec"}
		}
	}
	log.Debug("Running %s in container. Equivalent command: docker run -it --rm -e %s -w %s -v %s:%s -u \"%s\" %s %s",
		target.Label, strings.Join(config.Env, " -e "), config.WorkingDir, targetTestDir, config.WorkingDir,
		config.User, config.Image, strings.Join(config.Cmd, " "))
	c, err := dockerClient.ContainerCreate(context.Background(), config, hostConfig, nil, "")
	if err != nil && docker.IsErrNotFound(err) {
		// Image doesn't exist, need to try to pull it.
		// N.B. This is where we would authenticate if needed. Right now we are not doing anything.
		state.LogBuildResult(tid, target.Label, core.TargetTesting, "Pulling image...")
		r, err := dockerClient.ImagePull(context.Background(), config.Image, types.ImagePullOptions{})
		if err != nil {
			return nil, fmt.Errorf("Failed to pull image: %s", err)
		}
		defer r.Close()
		// I assume we have to exhaust this Reader before continuing. The docs are not super clear on how we know at what point the pull has completed.
		if _, err := io.Copy(ioutil.Discard, r); err != nil {
			return nil, fmt.Errorf("Failed to pull image: %s", err)
		}
		state.LogBuildResult(tid, target.Label, core.TargetTesting, "Testing...")
		c, err = dockerClient.ContainerCreate(context.Background(), config, hostConfig, nil, "")
		if err != nil {
			return nil, fmt.Errorf("Failed to create container: %s", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("Failed to create container: %s", err)
	}
	for _, warning := range c.Warnings {
		log.Warning("%s creating container: %s", target.Label, warning)
	}
	defer func() {
		if err := dockerClient.ContainerStop(context.Background(), c.ID, nil); err != nil {
			log.Warning("Failed to stop container for %s: %s", target.Label, err)
			return // ContainerRemove will fail if it's not stopped.
		}
		if err := dockerClient.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			log.Warning("Failed to remove container for %s: %s", target.Label, err)
		}
	}()
	if err := dockerClient.ContainerStart(context.Background(), c.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("Failed to start container: %s", err)
	}

	timeout := target.TestTimeout
	if timeout == 0 {
		timeout = time.Duration(state.Config.Test.Timeout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	waitChan, errChan := dockerClient.ContainerWait(ctx, c.ID, container.WaitConditionNotRunning)
	var status int64
	select {
	case body := <-waitChan:
		status = body.StatusCode
	case err := <-errChan:
		return nil, fmt.Errorf("Container failed running: %s", err)
	}
	// Now retrieve the results and any other files.
	if !target.NoTestOutput {
		retrieveFile(state, target, c.ID, resultsFile, true)
	}
	if state.NeedCoverage {
		retrieveFile(state, target, c.ID, path.Join(testDir, "test.coverage"), false)
	}
	for _, output := range target.TestOutputs {
		retrieveFile(state, target, c.ID, path.Join(testDir, output), false)
	}
	r, err := dockerClient.ContainerLogs(context.Background(), c.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving container output: %s", err)
	} else if status != 0 {
		return b, fmt.Errorf("Exit code %d", status)
	}
	return b, nil
}

func runPossiblyContainerisedTest(tid int, state *core.BuildState, target *core.BuildTarget) (out []byte, err error) {
	if target.Containerise {
		if state.Config.Test.DefaultContainer == core.ContainerImplementationNone {
			log.Warning("Target %s specifies that it should be tested in a container, but test "+
				"containers are disabled in your .plzconfig.", target.Label)
			return runTest(state, target)
		}
		out, err = runContainerisedTest(tid, state, target)
		if err != nil && state.Config.Docker.AllowLocalFallback {
			log.Warning("Failed to run %s containerised: %s %s. Falling back to local version.",
				target.Label, out, err)
			return runTest(state, target)
		}
		return out, err
	}
	return runTest(state, target)
}

// retrieveFile retrieves a single file (or directory) from a Docker container.
func retrieveFile(state *core.BuildState, target *core.BuildTarget, cid string, filename string, warn bool) {
	if err := retrieveOneFile(state, target, cid, filename); err != nil {
		if warn {
			log.Warning("Failed to retrieve output for %s: %s", target.Label, err)
		} else {
			log.Debug("Failed to retrieve output for %s: %s", target.Label, err)
		}
	}
}

// retrieveOneFile retrieves a single file from a Docker container.
func retrieveOneFile(state *core.BuildState, target *core.BuildTarget, cid string, filename string) error {
	log.Debug("Attempting to retrieve file %s for %s...", filename, target.Label)
	r, _, err := dockerClient.CopyFromContainer(context.Background(), cid, filename)
	if err != nil {
		return err
	}
	defer r.Close()
	// Files come out as a tarball (this isn't documented but seems empirically true).
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return err
		} else if hdr.Mode&int64(os.ModeDir) != 0 || strings.HasSuffix(hdr.Name, "/") {
			continue // Don't do anything specific with directories, only the files in them.
		}
		out := path.Join(target.TestDir(), hdr.Name)
		if err := os.MkdirAll(path.Dir(out), core.DirPermissions); err != nil {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			return err
		}
	}
	return nil
}
