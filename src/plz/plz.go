package plz

import (
	"path"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/metrics"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/remote"
	"github.com/thought-machine/please/src/test"
)

var log = logging.MustGetLogger("plz")

// Run runs a build to completion.
// The given state object controls most of the parameters to it and can be interrogated
// afterwards to find success / failure.
// To get detailed results as it runs, use state.Results. You should call that *before*
// starting this (otherwise a sufficiently fast build may bypass you completely).
func Run(targets, preTargets []core.BuildLabel, state *core.BuildState, config *core.Configuration, arch cli.Arch) {
	parse.InitParser(state)
	build.Init(state)
	if state.Config.Remote.URL != "" {
		state.RemoteClient = remote.New(state)
	}
	if config.Display.SystemStats {
		go state.UpdateResources()
	}

	// Start looking for the initial targets to kick the build off
	go findOriginalTasks(state, preTargets, targets, arch)

	parses, builds, remoteBuilds, tests, remoteTests := state.TaskQueues()

	// Start up all the build workers
	var wg sync.WaitGroup
	wg.Add(1 + config.Please.NumThreads + config.NumRemoteExecutors())
	go func() {
		// All parses happen async on separate goroutines so we don't have to worry about them blocking.
		// They manage concurrency control themselves.
		for task := range parses {
			go func(task core.ParseTask) {
				parse.Parse(0, state, task.Label, task.Dependent, task.ForSubinclude)
				state.TaskDone()
			}(task)
		}
		wg.Done()
	}()
	for i := 0; i < config.Please.NumThreads; i++ {
		go func(tid int) {
			doTasks(tid, state, builds, tests, false)
			wg.Done()
		}(i)
	}
	for i := 0; i < config.NumRemoteExecutors(); i++ {
		go func(tid int) {
			doTasks(tid, state, remoteBuilds, remoteTests, true)
			wg.Done()
		}(config.Please.NumThreads + i)
	}
	// Wait until they've all exited, which they'll do once they have no tasks left.
	wg.Wait()
	if state.Cache != nil {
		state.Cache.Shutdown()
	}
	if state.RemoteClient != nil {
		_, _, in, out := state.RemoteClient.DataRate()
		log.Info("Total remote RPC data in: %d out: %d", in, out)
	}
	state.CloseResults()
	metrics.Push(config)
}

// RunHost is a convenience function that uses the host architecture, the given state's
// configuration and no pre targets. It is otherwise identical to Run.
func RunHost(targets []core.BuildLabel, state *core.BuildState) {
	Run(targets, nil, state, state.Config, cli.HostArch())
}

func doTasks(tid int, state *core.BuildState, builds <-chan core.BuildTask, tests <-chan core.TestTask, remote bool) {
	for builds != nil || tests != nil {
		select {
		case l, ok := <-builds:
			if !ok {
				builds = nil
				break
			}
			build.Build(tid, state, l, remote)
			state.TaskDone()
		case testTask, ok := <-tests:
			if !ok {
				tests = nil
				break
			}
			test.Test(tid, state, testTask.Label, remote, testTask.Run)
			state.TaskDone()
		}
	}
}

// findOriginalTasks finds the original parse tasks for the original set of targets.
func findOriginalTasks(state *core.BuildState, preTargets, targets []core.BuildLabel, arch cli.Arch) {
	if state.Config.Bazel.Compatibility && fs.FileExists("WORKSPACE") {
		// We have to parse the WORKSPACE file before anything else to understand subrepos.
		// This is a bit crap really since it inhibits parallelism for the first step.
		parse.Parse(0, state, core.NewBuildLabel("workspace", "all"), core.OriginalTarget, false)
	}
	if arch.Arch != "" {
		// Set up a new subrepo for this architecture.
		state.Graph.AddSubrepo(core.SubrepoForArch(state, arch))
	}
	if len(preTargets) > 0 {
		findOriginalTaskSet(state, preTargets, false, arch)
		for _, target := range preTargets {
			if target.IsAllTargets() {
				log.Debug("Waiting for pre-target %s...", target)
				state.SyncParsePackage(target)
				log.Debug("Pre-target %s parsed, continuing...", target)
			}
		}
		for _, target := range state.ExpandLabels(preTargets) {
			log.Debug("Waiting for pre-target %s...", target)
			state.WaitForTargetAndEnsureDownload(target, targets[0])
			log.Debug("Pre-target %s built, continuing...", target)
		}
	}
	findOriginalTaskSet(state, targets, true, arch)
	log.Debug("Original target scan complete")
	state.TaskDone() // initial target adding counts as one.
}

func findOriginalTaskSet(state *core.BuildState, targets []core.BuildLabel, addToList bool, arch cli.Arch) {
	for _, target := range ReadStdinLabels(targets) {
		findOriginalTask(state, target, addToList, arch)
	}
}

func findOriginalTask(state *core.BuildState, target core.BuildLabel, addToList bool, arch cli.Arch) {
	if arch != cli.HostArch() {
		target.Subrepo = arch.String()
	}
	if target.IsAllSubpackages() {
		// Any command-line labels with subrepos and ... require us to know where they are in order to
		// walk the directory tree, so we have to make sure the subrepo exists first.
		dir := target.PackageName
		prefix := ""
		if target.Subrepo != "" {
			state.WaitForTargetAndEnsureDownload(target.SubrepoLabel(), target)
			subrepo := state.Graph.SubrepoOrDie(target.Subrepo)
			dir = subrepo.Dir(dir)
			prefix = subrepo.Dir(prefix)
		}
		for filename := range FindAllBuildFiles(state.Config, dir, "") {
			dirname, _ := path.Split(filename)
			l := core.NewBuildLabel(strings.TrimLeft(strings.TrimPrefix(strings.TrimRight(dirname, "/"), prefix), "/"), "all")
			l.Subrepo = target.Subrepo
			state.AddOriginalTarget(l, addToList)
		}
	} else {
		state.AddOriginalTarget(target, addToList)
	}
}

// FindAllBuildFiles finds all BUILD files under a particular path.
// Used to implement rules with ... where we need to know all possible packages
// under that location.
func FindAllBuildFiles(config *core.Configuration, rootPath, prefix string) <-chan string {
	ch := make(chan string)
	go func() {
		if rootPath == "" {
			rootPath = "."
		}
		if err := fs.Walk(rootPath, func(name string, isDir bool) error {
			basename := path.Base(name)
			if basename == core.OutDir || (isDir && strings.HasPrefix(basename, ".") && name != ".") {
				return filepath.SkipDir // Don't walk output or hidden directories
			} else if isDir && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(prefix, name) {
				return filepath.SkipDir // Skip any directory without the prefix we're after (but not any directory beneath that)
			} else if config.IsABuildFile(basename) && !isDir {
				ch <- name
			} else if cli.ContainsString(name, config.Parse.ExperimentalDir) {
				return filepath.SkipDir // Skip the experimental directory if it's set
			}
			// Check against blacklist
			for _, dir := range config.Parse.BlacklistDirs {
				if dir == basename || strings.HasPrefix(name, dir) {
					return filepath.SkipDir
				}
			}
			return nil
		}); err != nil {
			log.Fatalf("Failed to walk tree under %s; %s\n", rootPath, err)
		}
		close(ch)
	}()
	return ch
}

// ReadingStdin returns true if any of the given build labels are reading from stdin.
func ReadingStdin(labels []core.BuildLabel) bool {
	for _, l := range labels {
		if l == core.BuildLabelStdin {
			return true
		}
	}
	return false
}

// ReadStdinLabels reads any of the given labels from stdin, if any of them indicate it
// (i.e. if ReadingStdin(labels) is true, otherwise it just returns them.
func ReadStdinLabels(labels []core.BuildLabel) []core.BuildLabel {
	if !ReadingStdin(labels) {
		return labels
	}
	ret := []core.BuildLabel{}
	for _, l := range labels {
		if l == core.BuildLabelStdin {
			for s := range cli.ReadStdin() {
				ret = append(ret, core.ParseBuildLabels([]string{s})...)
			}
		} else {
			ret = append(ret, l)
		}
	}
	return ret
}
