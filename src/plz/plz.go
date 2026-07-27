package plz

import (
	"context"
	"fmt"
	"iter"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/peterebden/go-cli-init/v5/flags"
	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/cmap"
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
func Run(targets, preTargets []core.BuildLabel, state *core.BuildState, progress *Progress, arch cli.Arch) error {
	build.Init(state)
	if state.Config.Remote.URL != "" {
		state.RemoteClient = remote.New(state)
	}
	if state.Config.Display.SystemStats {
		go state.UpdateResources()
	}

	parse.InitParser(state)

	localLimiter := make(limiter, state.Config.Please.NumThreads)
	remoteLimiter := make(limiter, state.Config.NumRemoteExecutors())
	anyRemote := state.Config.NumRemoteExecutors() > 0

	limiter := func(remote bool) limiter {
		if remote {
			return remoteLimiter
		}
		return localLimiter
	}

	// TODO(peter): we should probably stitch these contexts around more
	g, ctx := errgroup.WithContext(context.Background())

	state.Parse = func(label core.BuildLabel) (*core.Package, error) {
		if pkg, err := state.Graph.PackageOrWait(ctx, label); err != nil || pkg != nil {
			return pkg, err
		}
		// If we get here then we have to parse it (we only get here if we are the first one)
		progress.numParsing.Add(1)
		defer progress.numParsing.Add(-1)
		// If the target defines a subrepo, we must make sure that is built first.
		if label.Subrepo != "" {
			sl := label.SubrepoLabel(state)
			if sl.Subrepo == label.Subrepo && sl.PackageName == label.PackageName {
				// TODO(peter): Unsure if this is a legit case or not.
				return nil, fmt.Errorf("subinclude from within same package of a subrepo")
			}
			if _, err := state.Parse(sl); err != nil {
				return nil, err
			}
		}
		// TODO(peter): can we drop the dependent here?
		return parse.Parse(state, label, label)
	}

	// parseTarget returns a particular build target, parsing the build file along the way if necessary.
	parseTarget := func(label core.BuildLabel) (*core.BuildTarget, error) {
		if target := state.Graph.Target(label); target != nil {
			return target, nil
		}
		pkg, err := state.Parse(label)
		if err != nil {
			return nil, err
		}
		if target := pkg.Target(label.Name); target != nil {
			return target, nil
		}
		return nil, fmt.Errorf("Parsed build file %s but it doesn't contain target %s%s", pkg.Filename, label.Name, pkg.SuggestTargets(label, label))
	}

	// resolveTarget resolves a target, dealing with require/provide as needed.
	resolveTarget := func(label core.BuildLabel, dependent *core.BuildTarget) iter.Seq2[*core.BuildTarget, error] {
		return func(yield func(*core.BuildTarget, error) bool) {
			target, err := parseTarget(label)
			if err != nil {
				yield(nil, err)
				return
			}
			// TODO(peter): We might want the minor optimisation here to avoid creating a slice in the common case
			provided := target.ProvideFor(dependent)
			if len(provided) == 1 && provided[0] == target.Label {
				yield(target, nil)
				return
			}
			// TODO(peter): Would parallelism here be useful?
			for _, p := range provided {
				if !yield(parseTarget(p)) {
					break
				}
			}
		}
	}

	buildDep := func(dep core.BuildLabel, target *core.BuildTarget) error {
		for t, err := range resolveTarget(dep, target) {
			if err != nil {
				return err
			}
			if _, err := state.Build(t.Label); err != nil {
				return err
			}
		}
		return nil
	}

	reallyBuild := func(target *core.BuildTarget) error {
		// TODO(peterebden): want to restructure how all this stuff sits on build targets as well
		g, _ := errgroup.WithContext(ctx)
		for dep := range target.BuildDependencyLabels() {
			g.Go(func() error {
				return buildDep(dep, target)
			})
		}
		for _, src := range target.AllSources() {
			if l, ok := src.Label(); ok {
				g.Go(func() error {
					return buildDep(l, target)
				})
			}
		}
		if err := g.Wait(); err != nil {
			return err
		}

		// TODO(peter): Need to handle targets getting extra deps added by post-build functions here.

		// TODO(peter): Temporary, we are going to get rid of all this
		if err := target.ResolveDependencies(state.Graph); err != nil {
			return err
		}

		// Now we are ready to build this target. Grab a thread and get started.
		remote := anyRemote && !target.Local
		limiter := limiter(remote)
		limiter.Acquire()
		defer limiter.Release()
		return build.Build(state, target, remote)
	}

	buildAll := func(label core.BuildLabel) error {
		pkg, err := state.Parse(label)
		if err != nil {
			return err
		}
		g, _ := errgroup.WithContext(ctx)
		for _, target := range pkg.AllTargets() {
			g.Go(func() error {
				return reallyBuild(target)
			})
		}
		return g.Wait()
	}

	// We use this to ensure that we only build each target exactly once.
	buildOnce := cmap.NewErrMap[core.BuildLabel, *core.BuildTarget](cmap.DefaultShardCount, func(l core.BuildLabel) uint64 {
		return cmap.XXHashes(l.Subrepo, l.PackageName, l.Name)
	}, nil)

	state.Build = func(label core.BuildLabel) (_ *core.BuildTarget, err error) {
		return buildOnce.GetOrSet(label, func() (*core.BuildTarget, error) {
			if label.IsAllTargets() {
				return nil, buildAll(label)
			}
			progress.numTotal.Add(1)
			defer progress.numDone.Add(1)
			target, err := parseTarget(label)
			if err != nil {
				return nil, err
			}
			return target, reallyBuild(target)
		})
	}

	state.Test = func(label core.BuildLabel) error {
		progress.numTotal.Add(int64(state.NumTestRuns))
		target, err := parseTarget(label)
		if err != nil {
			return err
		}
		g, _ := errgroup.WithContext(ctx)
		g.Go(func() error {
			_, err := state.Build(label)
			return err
		})
		// TODO(peterebden): More restructuring here. It's sort of weird that this doesn't go through resolveTarget.
		for dep := range target.IterAllRuntimeDependencies(state.Graph) {
			g.Go(func() error {
				_, err := state.Build(dep)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
		// Now we're ready to test this target.
		// TODO(peter): Is it okay for none of these to return errors? I _think_ so and we will capture it later?
		remote := anyRemote && !target.Local
		limiter := limiter(remote)
		if state.TestSequentially || state.NumTestRuns == 1 { // minor optimisation to avoid creating unnecessary goroutines
			limiter.Acquire()
			defer limiter.Release()
			for run := range int(state.NumTestRuns) {
				test.Test(state, target, remote, run+1)
				progress.numDone.Add(1)
			}
			return nil
		}
		var wg sync.WaitGroup
		for run := range int(state.NumTestRuns) {
			wg.Go(func() {
				limiter.Acquire()
				defer limiter.Release()
				test.Test(state, target, remote, run+1)
				progress.numDone.Add(1)
			})
		}
		wg.Wait()
		return nil
	}

	// Register the preloaded targets with the parser
	if err := registerPreloads(state); err != nil {
		return err
	}

	// Start looking for the initial targets to kick the build off
	tf := taskFinder{
		state: state,
		arch:  arch,
		tasks: g,
	}
	if err := tf.FindOriginalTasks(preTargets, targets); err != nil {
		return err
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if state.Cache != nil {
		state.Cache.Shutdown()
	}
	if state.RemoteClient != nil {
		_, _, in, out := state.RemoteClient.DataRate()
		log.Info("Total remote RPC data in: %d out: %d", in, out)
	}
	state.CloseResults()
	metrics.Push(state.Config.Metrics, state.Config.IsRemoteExecution())
	return nil
}

// RunHost is a convenience function that uses the host architecture, the given state's
// configuration and no pre targets. It is otherwise identical to Run.
func RunHost(targets []core.BuildLabel, state *core.BuildState) {
	Run(targets, nil, state, &Progress{}, cli.HostArch())
}

type taskFinder struct {
	tasks *errgroup.Group
	state *core.BuildState
	arch  cli.Arch
}

// findOriginalTasks finds the original parse tasks for the original set of targets.
func (tf *taskFinder) FindOriginalTasks(preTargets, targets []core.BuildLabel) error {
	log.Debug("Original target scan beginning...")
	if tf.state.Config.Bazel.Compatibility && fs.FileExists("WORKSPACE") {
		// We have to parse the WORKSPACE file before anything else to understand subrepos.
		// This is a bit crap really since it inhibits parallelism for the first step.
		if _, err := tf.state.Parse(core.NewBuildLabel("workspace", "all")); err != nil {
			return err
		}
	}
	if tf.arch.Arch != "" && tf.arch != cli.HostArch() {
		// Set up a new subrepo for this architecture.
		tf.state.Graph.AddSubrepo(core.SubrepoForArch(tf.state, tf.arch))
	}
	if len(preTargets) > 0 {
		tf.findOriginalTaskSet(preTargets, false, true)
		if err := tf.tasks.Wait(); err != nil {
			return err
		}
		tf.tasks = &errgroup.Group{}
	}
	tf.findOriginalTaskSet(targets, tf.state.NeedTests, tf.state.NeedBuild)
	if tf.state.NeedDebugDeps {
		if len(targets) != 1 {
			return fmt.Errorf("expected exactly 1 target in debug mode; got %d", len(targets))
		}
		tf.tasks.Go(func() error {
			return tf.queueTargetsForDebug(targets[0])
		})
	}
	if err := tf.tasks.Wait(); err != nil {
		return err
	}
	log.Debug("Original target scan complete")
	return nil
}

func (tf *taskFinder) findOriginalTaskSet(targets []core.BuildLabel, needTest, needBuild bool) {
	for _, target := range ReadStdinLabels(targets) {
		tf.tasks.Go(func() error {
			return tf.findOriginalTask(target, needTest, needBuild)
		})
	}
}

func (tf *taskFinder) queueTargetsForDebug(target core.BuildLabel) error {
	if _, err := tf.state.Parse(target); err != nil {
		return err
	}
	t := tf.state.Graph.TargetOrDie(target)
	for _, tool := range t.AllDebugTools() {
		if l, ok := tool.Label(); ok {
			tf.findOriginalTask(l, false, true)
		}
	}
	for _, data := range t.AllDebugData() {
		if l, ok := data.Label(); ok {
			tf.findOriginalTask(l, false, true)
		}
	}
	return nil
}

func stripHostRepoName(config *core.Configuration, label core.BuildLabel) core.BuildLabel {
	if label.Subrepo == "" {
		return label
	}

	if label.Subrepo == config.PluginDefinition.Name {
		label.Subrepo = ""
		return label
	}
	label.Subrepo = strings.TrimPrefix(label.Subrepo, config.PluginDefinition.Name+"@")

	hostArch := cli.HostArch()
	if label.Subrepo == hostArch.String() {
		label.Subrepo = ""
	}
	label.Subrepo = strings.TrimSuffix(label.Subrepo, "@"+hostArch.String())

	return label
}

func (tf *taskFinder) findOriginalTask(target core.BuildLabel, needTest, needBuild bool) error {
	if tf.arch != cli.HostArch() {
		target = core.LabelToArch(target, tf.arch)
	}
	target = stripHostRepoName(tf.state.Config, target)
	if !target.IsAllSubpackages() {
		tf.queueTask(target, needTest, needBuild)
		return nil
	}
	// Any command-line labels with subrepos and ... require us to know where they are in order to
	// walk the directory tree, so we have to make sure the subrepo exists first.
	dir := target.PackageName
	prefix := ""
	if target.Subrepo != "" {
		subrepoLabel := target.SubrepoLabel(tf.state)
		if target, err := tf.state.Build(subrepoLabel); err != nil {
			return err
		} else if err := tf.state.EnsureDownloaded(target); err != nil {
			return err
		}
		// Targets now get activated during parsing, so can be built before we finish parsing their package.
		pkg, err := tf.state.Parse(subrepoLabel)
		if err != nil {
			return err
		}
		dir = pkg.Subrepo.Dir(dir)
		prefix = pkg.Subrepo.Dir(prefix)
	}
	for filename := range FindAllBuildFiles(tf.state.Config, dir, "") {
		dirname, _ := filepath.Split(filename)
		l := core.NewBuildLabel(strings.TrimLeft(strings.TrimPrefix(strings.TrimRight(dirname, "/"), prefix), "/"), "all")
		l.Subrepo = target.Subrepo
		tf.queueTask(l, needTest, needBuild)
	}
	return nil
}

func (tf *taskFinder) queueTask(target core.BuildLabel, needTest, needBuild bool) {
	tf.tasks.Go(func() error {
		if needTest {
			return tf.state.Test(target)
		} else if needBuild {
			_, err := tf.state.Build(target)
			// TODO(peter): Ensure this gets downloaded if needed
			return err
		}
		_, err := tf.state.Parse(target)
		return err
	})
}

// registerPreloads waits for all preloaded subinclude targets to be built, downloads them, and then registers them with
// the interpreter. We have to actually register them otherwise this will return before we build any
// transitive subincludes.
func registerPreloads(state *core.BuildState) error {
	var eg errgroup.Group
	for _, inc := range state.GetPreloadedSubincludes() {
		if inc.IsPseudoTarget() {
			return fmt.Errorf("Can't preload pseudotarget %v", inc)
		}

		// Queue them up asynchronously to feed the queues as quickly as possible
		eg.Go(func() error {
			if _, err := state.Build(inc); err != nil {
				return err
			}
			return state.Parser.RegisterPreload(inc)
		})
	}
	// We must wait for all the subinclude targets to be built otherwise updating the locals might race with parsing
	// a package
	return eg.Wait()
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
			ret = append(ret, ReadAndParseStdinLabels()...)
		} else {
			ret = append(ret, l)
		}
	}
	return ret
}

// ReadAndParseStdinLabels unconditionally reads stdin and parses it into build labels.
func ReadAndParseStdinLabels() []core.BuildLabel {
	ret := []core.BuildLabel{}
	for s := range flags.ReadStdin() {
		ret = append(ret, core.ParseBuildLabels([]string{s})...)
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

// Progress records some numerical progress in regard to tasks we have performed / yet to perform.
type Progress struct {
	numTotal, numDone, numParsing atomic.Int64
}

// NumTotal returns the total number of tasks for this execution.
// These are discovered as we go so this can increase over time.
func (p *Progress) NumTotal() int {
	return int(p.numTotal.Load())
}

// NumDone returns the number of tasks completed for this execution.
func (p *Progress) NumDone() int {
	return int(p.numDone.Load())
}

// NumParsing returns the number of BUILD files currently being parsed.
func (p *Progress) NumParsing() int {
	return int(p.numParsing.Load())
}
