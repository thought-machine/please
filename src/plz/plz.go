package plz

import (
	"strings"
	"sync"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/follow"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/test"
	"github.com/thought-machine/please/src/utils"
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

	if config.Events.Port != 0 && state.NeedBuild {
		shutdown := follow.InitialiseServer(state, config.Events.Port)
		defer shutdown()
	}
	if config.Events.Port != 0 || config.Display.SystemStats {
		go follow.UpdateResources(state)
	}

	// Start looking for the initial targets to kick the build off
	go findOriginalTasks(state, preTargets, targets, arch)

	parses, builds, tests, remoteBuilds, remoteTests := state.TaskQueues()

	// Start up all the build workers
	var wg sync.WaitGroup
	wg.Add(config.Please.NumThreads + config.Remote.NumExecutors)
	for i := 0; i < config.Please.NumThreads; i++ {
		go func(tid int) {
			doTasks(tid, state, parses, builds, tests, arch, false)
			wg.Done()
		}(i)
	}
	for i := 0; i < config.Remote.NumExecutors; i++ {
		go func(tid int) {
			doTasks(tid, state, nil, remoteBuilds, remoteTests, arch, true)
			wg.Done()
		}(config.Please.NumThreads + i)
	}
	// Wait until they've all exited, which they'll do once they have no tasks left.
	wg.Wait()
	if state.Cache != nil {
		state.Cache.Shutdown()
	}
}

// RunHost is a convenience function that uses the host architecture, the given state's
// configuration and no pre targets. It is otherwise identical to Run.
func RunHost(targets []core.BuildLabel, state *core.BuildState) {
	Run(targets, nil, state, state.Config, cli.Arch{})
}

func doTasks(tid int, state *core.BuildState, parses <-chan core.LabelPair, builds, tests <-chan core.BuildLabel, arch cli.Arch, remote bool) {
	for parses != nil || builds != nil || tests != nil {
		select {
		case p, ok := <-parses:
			if !ok {
				parses = nil
				break
			}
			state.ParsePool <- func() {
				parse.Parse(tid, state, p.Label, p.Dependent, p.ForSubinclude)
				state.TaskDone(false)
			}
		case l, ok := <-builds:
			if !ok {
				builds = nil
				break
			}
			build.Build(tid, state, l, remote)
			state.TaskDone(true)
		case l, ok := <-tests:
			if !ok {
				tests = nil
				break
			}
			test.Test(tid, state, l, remote)
			state.TaskDone(true)
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
				state.WaitForPackage(target)
				log.Debug("Pre-target %s parsed, continuing...", target)
			}
		}
		for _, target := range state.ExpandLabels(preTargets) {
			log.Debug("Waiting for pre-target %s...", target)
			state.WaitForBuiltTarget(target, targets[0])
			log.Debug("Pre-target %s built, continuing...", target)
		}
	}
	findOriginalTaskSet(state, targets, true, arch)
	state.TaskDone(true) // initial target adding counts as one.
}

func findOriginalTaskSet(state *core.BuildState, targets []core.BuildLabel, addToList bool, arch cli.Arch) {
	for _, target := range targets {
		if target == core.BuildLabelStdin {
			for label := range cli.ReadStdin() {

				findOriginalTask(state, core.ParseBuildLabels([]string{label})[0], addToList, arch)
			}
		} else {
			findOriginalTask(state, target, addToList, arch)
		}
	}
}

func findOriginalTask(state *core.BuildState, target core.BuildLabel, addToList bool, arch cli.Arch) {
	if arch.Arch != "" {
		target.Subrepo = arch.String()
	}
	if target.IsAllSubpackages() {
		// Any command-line labels with subrepos and ... require us to know where they are in order to
		// walk the directory tree, so we have to make sure the subrepo exists first.
		dir := target.PackageName
		prefix := ""
		if target.Subrepo != "" {
			state.WaitForBuiltTarget(target.SubrepoLabel(), target)
			subrepo := state.Graph.SubrepoOrDie(target.Subrepo)
			dir = subrepo.Dir(dir)
			prefix = subrepo.Dir(prefix)
		}
		for pkg := range utils.FindAllSubpackages(state.Config, dir, "") {
			l := core.NewBuildLabel(strings.TrimLeft(strings.TrimPrefix(pkg, prefix), "/"), "all")
			l.Subrepo = target.Subrepo
			state.AddOriginalTarget(l, addToList)
		}
	} else {
		state.AddOriginalTarget(target, addToList)
	}
}
