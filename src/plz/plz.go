package plz

import (
	"path/filepath"
	"strings"

	"github.com/peterebden/go-cli-init/v5/flags"
	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/metrics"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/remote"
	"github.com/thought-machine/please/src/test"
)

var log = logging.Log

// Run runs a build to completion.
// The given state object controls most of the parameters to it and can be interrogated
// afterwards to find success / failure.
// To get detailed results as it runs, use state.Results. You should call that *before*
// starting this (otherwise a sufficiently fast build may bypass you completely).
func Run(targets, preTargets []core.BuildLabel, state *core.BuildState, config *core.Configuration, arch cli.Arch) {
	build.Init(state)
	if state.Config.Remote.URL != "" {
		state.RemoteClient = remote.New(state)
	}
	if config.Display.SystemStats {
		go state.UpdateResources()
	}

	parse.InitParser(state)

	parses, actions := state.TaskQueues()

	localLimiter := make(limiter, config.Please.NumThreads)
	remoteLimiter := make(limiter, config.NumRemoteExecutors())
	anyRemote := config.NumRemoteExecutors() > 0

	// Start up all the build workers
	var g errgroup.Group
	g.Go(func() error {
		return findOriginalTasks(state, preTargets, targets, arch)
	})
	g.Go(func() error {
		for task := range parses {
			go func(task core.ParseTask) {
				parse.Parse(state, task.Label, task.Dependent, task.Mode)
				state.TaskDone()
			}(task)
		}
		return nil
	})
	g.Go(func() error {
		for task := range actions {
			go func(task core.Task) {
				remote := anyRemote && !task.Target.Local
				if remote {
					remoteLimiter.Acquire()
					defer remoteLimiter.Release()
				} else {
					localLimiter.Acquire()
					defer localLimiter.Release()
				}
				switch task.Type {
				case core.TestTask:
					test.Test(state, task.Target, remote, int(task.Run))
				case core.BuildTask:
					build.Build(state, task.Target, remote)
				}
				state.TaskDone()
			}(task)
		}
		return nil
	})
	// Wait until they've all exited, which they'll do once they have no tasks left.
	err := g.Wait()
	if state.Cache != nil {
		state.Cache.Shutdown()
	}
	if state.RemoteClient != nil {
		_, _, in, out := state.RemoteClient.DataRate()
		log.Info("Total remote RPC data in: %d out: %d", in, out)
	}
	state.CloseResults()
	metrics.Push(config)
	if err != nil {
		log.Fatalf("%s", err)
	}
}

// RunHost is a convenience function that uses the host architecture, the given state's
// configuration and no pre targets. It is otherwise identical to Run.
func RunHost(targets []core.BuildLabel, state *core.BuildState) {
	Run(targets, nil, state, state.Config, cli.HostArch())
}

// findOriginalTasks finds the original parse tasks for the original set of targets.
func findOriginalTasks(state *core.BuildState, preTargets, targets []core.BuildLabel, arch cli.Arch) error {
	var g errgroup.Group
	if state.Config.Bazel.Compatibility && fs.FileExists("WORKSPACE") {
		// We have to parse the WORKSPACE file before anything else to understand subrepos.
		// This is a bit crap really since it inhibits parallelism for the first step.
		parse.Parse(state, core.NewBuildLabel("workspace", "all"), core.OriginalTarget, core.ParseModeNormal)
	}
	if arch.Arch != "" && arch != cli.HostArch() {
		// Set up a new subrepo for this architecture.
		state.Graph.AddSubrepo(core.SubrepoForArch(state, arch))
	}
	if len(preTargets) > 0 {
		findOriginalTaskSet(state, &g, preTargets, false, arch)
		for _, target := range preTargets {
			if target.IsAllTargets() {
				log.Debug("Waiting for pre-target %s...", target)
				state.SyncParsePackage(target)
				log.Debug("Pre-target %s parsed, continuing...", target)
			}
		}
		for _, target := range state.ExpandLabels(preTargets) {
			log.Debug("Waiting for pre-target %s...", target)
			state.WaitForTargetAndEnsureDownload(target, targets[0], core.ParseModeNormal)
			log.Debug("Pre-target %s built, continuing...", target)
		}
	}
	findOriginalTaskSet(state, &g, targets, true, arch)
	if err := g.Wait(); err != nil {
		log.Debug("Original target scan failed: %s", err)
		state.Stop()
		return err
	}
	log.Debug("Original target scan complete")
	state.TaskDone() // initial target adding counts as one.
	return nil
}

func findOriginalTaskSet(state *core.BuildState, g *errgroup.Group, targets []core.BuildLabel, addToList bool, arch cli.Arch) {
	for _, target := range ReadStdinLabels(targets) {
		findOriginalTask(state, g, target, addToList, arch)
	}
}

func stripHostRepoName(config *core.Configuration, label core.BuildLabel) core.BuildLabel {
	if label.Subrepo == "" {
		return label
	}

	if label.Subrepo == config.PluginDefinition.Name {
		label.Subrepo = ""
		return label
	}
	label.Subrepo = strings.TrimPrefix(label.Subrepo, config.PluginDefinition.Name+"_")

	hostArch := cli.HostArch()
	if label.Subrepo == hostArch.String() {
		label.Subrepo = ""
	}
	label.Subrepo = strings.TrimSuffix(label.Subrepo, "_"+hostArch.String())

	return label
}

func findOriginalTask(state *core.BuildState, g *errgroup.Group, target core.BuildLabel, addToList bool, arch cli.Arch) {
	if arch != cli.HostArch() {
		target = core.LabelToArch(target, arch)
	}
	target = stripHostRepoName(state.Config, target)
	if target.IsAllSubpackages() {
		// Any command-line labels with subrepos and ... require us to know where they are in order to
		// walk the directory tree, so we have to make sure the subrepo exists first.
		dir := target.PackageName
		prefix := ""
		if target.Subrepo != "" {
			subrepoLabel := target.SubrepoLabel(state, "")
			state.WaitForTargetAndEnsureDownload(subrepoLabel, target, core.ParseModeNormal)
			// Targets now get activated during parsing, so can be built before we finish parsing their package.
			state.WaitForPackage(subrepoLabel, target, core.ParseModeNormal)
			subrepo := state.Graph.SubrepoOrDie(target.Subrepo)
			dir = subrepo.Dir(dir)
			prefix = subrepo.Dir(prefix)
		}
		for filename := range FindAllBuildFiles(state.Config, dir, "") {
			dirname, _ := filepath.Split(filename)
			l := core.NewBuildLabel(strings.TrimLeft(strings.TrimPrefix(strings.TrimRight(dirname, "/"), prefix), "/"), "all")
			l.Subrepo = target.Subrepo
			queueTarget(state, g, l, addToList)
		}
	} else {
		queueTarget(state, g, target, addToList)
	}
}

// queueTarget marks a target as an original target and enqueues it for building, testing, or whatever.
func queueTarget(state *core.BuildState, g *errgroup.Group, label core.BuildLabel, addToList bool) {
	g.Go(func() error {
		if addToList {
			state.AddOriginalTarget(label)
		}
		if label.IsAllTargets() {
			for _, target := range state.WaitForPackage(label, core.OriginalTarget, core.ParseModeNormal).AllTargets() {
				queueTarget(state, g, target.Label, false)
			}
			return nil
		} else if !state.NeedBuild && !state.NeedTests && !state.ParsePackageOnly {
			_, err := state.ParseTree(label, core.OriginalTarget)
			return err
		} else if target, err := state.Parse(label, core.OriginalTarget, core.ParseModeNormal); err != nil || state.ParsePackageOnly {
			return err
		} else if !state.NeedTests {
			return state.Build(target, core.ParseModeNormal)
		} else {
			return state.Test(target)
		}
	})
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
			basename := filepath.Base(name)
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
			for s := range flags.ReadStdin() {
				ret = append(ret, core.ParseBuildLabels([]string{s})...)
			}
		} else {
			ret = append(ret, l)
		}
	}
	return ret
}

// A limiter allows only a certain number of concurrent tasks
// TODO(peterebden): We have about four of these now, commonise this somewhere
type limiter chan struct{}

func (l limiter) Acquire() {
	l <- struct{}{}
}

func (l limiter) Release() {
	<-l
}
