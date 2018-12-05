package plz

import (
	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/follow"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/metrics"
	"github.com/thought-machine/please/src/output"
	"github.com/thought-machine/please/src/parse"
	"sync"
	"github.com/thought-machine/please/src/test"
	"github.com/thought-machine/please/src/utils"
)

// InitOpts represents initialization options for please. These are usually being passed as cli args
type InitOpts struct {
	ParsePackageOnly bool
	VisibilityParse  bool
	DetailedTests    bool

	// Don't stop on first failed target
	KeepGoing    bool
	PrettyOutput bool
	ShouldRun    bool
	// Show status of each target in output after build
	ShowStatus bool

	TraceFile string
	Arch      cli.Arch
	NoLock    bool
}

// Init initialized the build for please
func Init(targets []core.BuildLabel, state *core.BuildState, config *core.Configuration, initOpts InitOpts) (bool, *core.BuildState) {
	parse.InitParser(state)
	build.Init(state)

	if config.Events.Port != 0 && state.NeedBuild {
		shutdown := follow.InitialiseServer(state, config.Events.Port)
		defer shutdown()
	}
	if config.Events.Port != 0 || config.Display.SystemStats {
		go follow.UpdateResources(state)
	}
	metrics.InitFromConfig(config)

	// Start looking for the initial targets to kick the build off
	go initOpts.findOriginalTasks(state, targets)
	// Start up all the build workers
	var wg sync.WaitGroup
	wg.Add(config.Please.NumThreads)
	for i := 0; i < config.Please.NumThreads; i++ {
		go func(tid int) {
			initOpts.doTasks(tid, state, state.Include, state.Exclude)
			wg.Done()
		}(i)
	}

	// Wait until they've all exited, which they'll do once they have no tasks left.
	go func() {
		wg.Wait()
		close(state.Results) // This will signal MonitorState (below) to stop.
	}()

	// Draw stuff to the screen while there are still results coming through.
	// TODO(bnm): Definitely refactor this at some point
	success := output.MonitorState(state, config.Please.NumThreads,
		!initOpts.PrettyOutput, initOpts.PrettyOutput, state.NeedBuild, state.NeedTests,
		initOpts.ShouldRun, initOpts.ShowStatus, initOpts.DetailedTests,
		initOpts.TraceFile)

	if state.Cache != nil {
		state.Cache.Shutdown()
	}

	return success, state
}

// InitDefault initializing please as a default set, this is used in textDocument/references call in langserver
func InitDefault(targets []core.BuildLabel, state *core.BuildState, config *core.Configuration) (bool, *core.BuildState) {
	initOpts := InitOpts{
		ParsePackageOnly: true,
		VisibilityParse:  true,
		DetailedTests:    false,
		KeepGoing:        false,
		ShouldRun:        false,
		ShowStatus:       false,
		TraceFile:        "",
		NoLock:           true,
	}
	return Init(targets, state,
		config, initOpts)
}

func (i *InitOpts) doTasks(tid int, state *core.BuildState, include, exclude []string) {
	for {
		label, dependor, t := state.NextTask()
		switch t {
		case core.Stop, core.Kill:
			return
		case core.Parse, core.SubincludeParse:
			t := t
			label := label
			dependor := dependor
			state.ParsePool <- func() {
				parse.Parse(tid, state, label, dependor, i.ParsePackageOnly, include, exclude, t == core.SubincludeParse)
				if i.VisibilityParse && state.IsOriginalTarget(label) {
					i.parseForVisibleTargets(state, label)
				}
				state.TaskDone(false)
			}
		case core.Build, core.SubincludeBuild:
			build.Build(tid, state, label)
			state.TaskDone(true)
		case core.Test:
			test.Test(tid, state, label)
			state.TaskDone(true)
		}
	}
}

// findOriginalTasks finds the original parse tasks for the original set of targets.
func (i *InitOpts) findOriginalTasks(state *core.BuildState, targets []core.BuildLabel) {
	if state.Config.Bazel.Compatibility && fs.FileExists("WORKSPACE") {
		// We have to parse the WORKSPACE file before anything else to understand subrepos.
		// This is a bit crap really since it inhibits parallelism for the first step.
		parse.Parse(0, state, core.NewBuildLabel("workspace", "all"), core.OriginalTarget,
			false, state.Include, state.Exclude, false)
	}
	if i.Arch.Arch != "" {
		// Set up a new subrepo for this architecture.
		state.Graph.AddSubrepo(core.SubrepoForArch(state, i.Arch))
	}
	for _, target := range targets {
		if target == core.BuildLabelStdin {
			for label := range cli.ReadStdin() {

				i.findOriginalTask(state, core.ParseBuildLabels([]string{label})[0], true)
			}
		} else {
			i.findOriginalTask(state, target, true)
		}
	}
	state.TaskDone(true) // initial target adding counts as one.
}

func (i *InitOpts) findOriginalTask(state *core.BuildState, target core.BuildLabel, addToList bool) {
	if i.Arch.Arch != "" {
		target.Subrepo = i.Arch.String()
	}
	if target.IsAllSubpackages() {
		for pkg := range utils.FindAllSubpackages(state.Config, target.PackageName, "") {
			state.AddOriginalTarget(core.NewBuildLabel(pkg, "all"), addToList)
		}
	} else {
		state.AddOriginalTarget(target, addToList)
	}
}

// parseForVisibleTargets adds parse tasks for any targets that the given label is visible to.
func (i *InitOpts) parseForVisibleTargets(state *core.BuildState, label core.BuildLabel) {
	if target := state.Graph.Target(label); target != nil {
		for _, vis := range target.Visibility {
			i.findOriginalTask(state, vis, false)
		}
	}
}
