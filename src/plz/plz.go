package plz

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"path/filepath"
	"slices"
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
	"github.com/thought-machine/please/src/parse/asp"
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

	parser := parse.InitParser(state)

	// This must happen however we exit; anything reading state.Results() (e.g. the display)
	// waits for that channel to be closed, so it would hang forever if we returned an error first.
	defer func() {
		if state.Cache != nil {
			state.Cache.Shutdown()
		}
		if state.RemoteClient != nil {
			_, _, in, out := state.RemoteClient.DataRate()
			log.Info("Total remote RPC data in: %d out: %d", in, out)
		}
		state.CloseResults()
		metrics.Push(state.Config.Metrics, state.Config.IsRemoteExecution())
	}()

	ctx, cancel := context.WithCancel(context.Background())
	state.Cancel = cancel

	r := runner{
		state:    state,
		arch:     arch,
		progress: progress,
		buildOnce: cmap.NewErrMap[core.BuildLabel, *core.BuildTarget](cmap.DefaultShardCount, func(l core.BuildLabel) uint64 {
			return cmap.XXHashes(l.Subrepo, l.PackageName, l.Name)
		}, nil),
		parseOnce: cmap.New[core.BuildLabel, struct{}](cmap.DefaultShardCount, func(l core.BuildLabel) uint64 {
			return cmap.XXHashes(l.Subrepo, l.PackageName, l.Name)
		}),
		localLimiter:  make(limiter, state.Config.Please.NumThreads),
		remoteLimiter: make(limiter, state.Config.NumRemoteExecutors()),
		anyRemote:     state.Config.NumRemoteExecutors() > 0,
	}
	g, ctx := r.group(ctx)
	r.tasks = g

	// We don't have context as an argument to this, because they're not fully plumbed through (but probably should be)
	state.Build = func(label, dependent core.BuildLabel) (*core.BuildTarget, error) {
		target, err := r.Build(ctx, label, dependent)
		if err != nil {
			return nil, err
		}
		// Anything calling this will likely want this thing to end up being downloaded (it's mostly for subincludes)
		return target, state.EnsureDownloaded(target)
	}
	state.Parse = func(label, dependent core.BuildLabel) (*core.Package, error) {
		return r.Parse(ctx, label, dependent)
	}

	// Register the preloaded targets with the parser
	if err := r.RegisterPreloads(ctx, state, parser); err != nil {
		return err
	}

	if state.Config.Bazel.Compatibility && fs.FileExists("WORKSPACE") {
		// We have to parse the WORKSPACE file before anything else to understand subrepos.
		// This is a bit crap really since it inhibits parallelism for the first step.
		if _, err := r.Parse(ctx, core.NewBuildLabel("workspace", "all"), core.OriginalTarget); err != nil {
			return err
		}
	}
	if arch.Arch != "" && arch != cli.HostArch() {
		// Set up a new subrepo for this architecture.
		state.Graph.AddSubrepo(core.SubrepoForArch(state, arch))
	}
	if len(preTargets) > 0 {
		r.FindOriginalTaskSet(ctx, preTargets, false, true)
		if err := g.Wait(); err != nil {
			return err
		}
		// Reset the group & context for next time
		ctx, cancel = context.WithCancel(context.Background())
		g, ctx = r.group(ctx)
		state.Cancel = cancel
		r.tasks = g
	}
	r.FindOriginalTaskSet(ctx, targets, r.state.NeedTests, r.state.NeedBuild)
	if state.NeedDebugDeps {
		if len(targets) != 1 {
			return fmt.Errorf("expected exactly 1 target in debug mode; got %d", len(targets))
		}
		g.Go(func() error {
			return r.queueTargetsForDebug(ctx, targets[0])
		})
	}

	return g.Wait()
}

// RunHost is a convenience function that uses the host architecture, the given state's
// configuration and no pre targets. It is otherwise identical to Run.
func RunHost(targets []core.BuildLabel, state *core.BuildState) {
	Run(targets, nil, state, &Progress{}, cli.HostArch())
}

type runner struct {
	tasks         *errgroup.Group
	state         *core.BuildState
	arch          cli.Arch
	progress      *Progress
	buildOnce     *cmap.ErrMap[core.BuildLabel, *core.BuildTarget]
	parseOnce     *cmap.Map[core.BuildLabel, struct{}]
	localLimiter  limiter
	remoteLimiter limiter
	anyRemote     bool
}

// Parse parses for a target. It can be called more than once for the same build label.
// The dependent is whatever is asking for this to be parsed; it's used to produce better error
// messages, and to detect a package that is asking to parse itself.
func (r *runner) Parse(ctx context.Context, label, dependent core.BuildLabel) (*core.Package, error) {
	return r.parse(ctx, label, dependent, false)
}

// tryParse is like Parse but doesn't report failures. It's used where a failure isn't necessarily an
// error, i.e. when we're speculatively looking for the package that might define a subrepo; the caller
// is responsible for reporting anything it can't handle itself.
func (r *runner) tryParse(ctx context.Context, label, dependent core.BuildLabel) (*core.Package, error) {
	return r.parse(ctx, label, dependent, true)
}

func (r *runner) parse(ctx context.Context, label, dependent core.BuildLabel, quiet bool) (*core.Package, error) {
	return r.state.Graph.GetOrSetPackage(ctx, label, func() (*core.Package, error) {
		r.progress.numParsing.Add(1)
		defer r.progress.numParsing.Add(-1)
		pkg, err := func() (*core.Package, error) {
			// If the target is in a subrepo that we don't know about yet, we must make sure that is defined first.
			// If we already have it there's nothing to do here; it's been registered by whatever parse defined it.
			if label.Subrepo != "" && r.state.Graph.Subrepo(label.Subrepo) == nil {
				if err := r.ensureSubrepo(ctx, label, dependent); err != nil {
					return nil, err
				}
			}
			return parse.Parse(r.state, label, dependent)
		}()
		if err != nil && !quiet {
			r.state.LogBuildError(label, core.ParseFailed, err, "Failed to parse package")
		}
		return pkg, err
	})
}

// ensureSubrepo makes sure that the subrepo the given label is in has been defined.
//
// A name like `linux_amd64` is ambiguous: it could be a subrepo defined by a target somewhere, or one
// of the architecture subrepos, which are implicitly defined and so have no defining target anywhere.
// We resolve that by always preferring a real definition, and only falling back to the architecture
// interpretation once we know there isn't one.
func (r *runner) ensureSubrepo(ctx context.Context, label, dependent core.BuildLabel) error {
	sl := label.SubrepoLabel(r.state)
	// The subrepo would be defined by a target in the dependent's package, which means that package is
	// the one currently being parsed - and since we didn't find the subrepo, the call that defines it
	// hasn't been reached yet. We can't wait for that parse because we are that parse.
	if inSamePackage(sl, dependent) {
		return fmt.Errorf("subrepo %v is not defined in this package yet. It must appear before it is used by %v", label.Subrepo, dependent)
	}
	// Parsing the package that should define it registers the subrepo as a side effect. A missing BUILD
	// file isn't fatal yet; that's exactly what we'd expect for an architecture subrepo.
	_, err := r.tryParse(ctx, sl, label)
	if err != nil && !errors.Is(err, parse.ErrMissingBuildFile) {
		return err
	}
	if r.state.Graph.Subrepo(label.Subrepo) != nil {
		return nil // The parse above defined it, we're done.
	}
	// Nothing defines it, so the only remaining possibility is an architecture subrepo.
	if arch, ok := couldBeArch(label.Subrepo); ok {
		r.state.Graph.MaybeAddSubrepo(core.SubrepoForArch(r.state, arch))
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("Subrepo %s is not defined (referenced by %s)", label.Subrepo, dependent)
}

// group returns an errgroup to run a set of tasks in, and the context to run them with.
//
// Normally that context is cancelled as soon as any of them fails, which stops us starting more work.
// With --keep_going we don't cancel anything, so everything that can still be built gets built; the
// group waits for all of it either way and Wait still returns the first error.
func (r *runner) group(ctx context.Context) (*errgroup.Group, context.Context) {
	if r.state.KeepGoing {
		return &errgroup.Group{}, ctx
	}
	return errgroup.WithContext(ctx)
}

// inSamePackage returns true if the two labels are in the same package (and hence, if one of them is
// currently being parsed, both are).
func inSamePackage(label, dependent core.BuildLabel) bool {
	return !dependent.IsOriginalTarget() && label.Subrepo == dependent.Subrepo && label.PackageName == dependent.PackageName
}

// RecursiveParse is like Parse but recurses down into all dependencies of the target as well.
func (r *runner) RecursiveParse(ctx context.Context, label, dependent core.BuildLabel) error {
	if !label.IsAllTargets() {
		return r.recursiveParse(ctx, label, dependent)
	}
	pkg, err := r.Parse(ctx, label, dependent)
	if err != nil {
		return err
	}
	g, gctx := r.group(ctx)
	for _, target := range pkg.AllTargets() {
		for dep := range target.DeclaredDependencies() {
			g.Go(func() error {
				// N.B. No need to deduplicate these; recursiveParse does that for the whole walk.
				return r.recursiveParse(gctx, dep, target.Label)
			})
		}
	}
	return g.Wait()
}

// recursiveParse parses a target and, transitively, everything it depends on.
func (r *runner) recursiveParse(ctx context.Context, label, dependent core.BuildLabel) error {
	if !r.parseOnce.Add(label, struct{}{}) {
		return nil // Someone else has this one; they're in the same errgroup so we needn't wait for them.
	}
	target, err := r.parseTarget(ctx, label, dependent)
	if err != nil {
		return err
	}
	g, gctx := r.group(ctx)
	for dep := range target.DeclaredDependencies() {
		g.Go(func() error {
			return r.recursiveParse(gctx, dep, target.Label)
		})
	}
	return g.Wait()
}

func (r *runner) parseTarget(ctx context.Context, label, dependent core.BuildLabel) (*core.BuildTarget, error) {
	if target := r.state.Graph.Target(label); target != nil {
		return target, nil
	}
	pkg, err := r.Parse(ctx, label, dependent)
	if err != nil {
		return nil, err
	}
	if target := pkg.Target(label.Name); target != nil {
		return target, nil
	}
	err = fmt.Errorf("Parsed build file %s but it doesn't contain target %s%s", pkg.Filename, label.Name, pkg.SuggestTargets(label, dependent))
	r.state.LogBuildError(label, core.ParseFailed, err, "%s", err)
	return nil, err
}

// resolveTarget resolves a target, dealing with require/provide as needed.
func (r *runner) resolveTarget(ctx context.Context, label core.BuildLabel, dependent *core.BuildTarget) iter.Seq2[*core.BuildTarget, error] {
	return func(yield func(*core.BuildTarget, error) bool) {
		target, err := r.parseTarget(ctx, label, dependent.Label)
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
			if !yield(r.parseTarget(ctx, p, dependent.Label)) {
				break
			}
		}
	}
}

// buildDep builds a single dependency of a target (which might of course turn into multiple when resolved)
func (r *runner) buildDep(ctx context.Context, dep core.BuildLabel, target *core.BuildTarget) error {
	for t, err := range r.resolveTarget(ctx, dep, target) {
		if err != nil {
			return err
		}
		if _, err := r.Build(ctx, t.Label, target.Label); err != nil {
			return err
		}
	}
	return nil
}

// buildOne builds a single target (which cannot be a pseudo-label like :all)
func (r *runner) buildOne(ctx context.Context, target *core.BuildTarget) error {
	g, gctx := r.group(ctx)
	for dep := range target.BuildDependencyLabels() {
		g.Go(func() error {
			return r.buildDep(gctx, dep, target)
		})
	}
	for _, src := range target.AllSources() {
		if l, ok := src.Label(); ok {
			g.Go(func() error {
				return r.buildDep(gctx, l, target)
			})
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}

	if target.ModifiedByCallback {
		// A pre- or post-build function modified this target post parse, so we need to check its dependencies again.
		g, gctx := r.group(ctx)
		for dep := range target.BuildDependencyLabels() {
			g.Go(func() error {
				return r.buildDep(gctx, dep, target)
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	// Okay, now the runtime dependencies can happen in parallel with the target itself.
	// N.B. Even when there are none we can't just build the target and return; its own callbacks
	//      can add some, which we won't know about until it's built.
	if deps := slices.Collect(target.RuntimeAndDataDependencies()); len(deps) == 0 {
		if err := r.buildJustOne(target); err != nil {
			return err
		}
	} else {
		g, gctx = r.group(ctx)
		g.Go(func() error {
			return r.buildJustOne(target)
		})
		for _, dep := range deps {
			g.Go(func() error {
				return r.buildDep(gctx, dep, target)
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	if !target.ModifiedByCallback {
		return nil
	}
	// It could have modified itself with its own post-build function, so we have to check runtime dpendencies again.
	// This is a little unfortunate that we can't immediately distinguish from the case we checked above.
	g, gctx = r.group(ctx)
	for dep := range target.RuntimeAndDataDependencies() {
		g.Go(func() error {
			return r.buildDep(gctx, dep, target)
		})
	}
	return g.Wait()
}

// buildJustOne calls the build for a single target.
func (r *runner) buildJustOne(target *core.BuildTarget) error {
	remote := r.anyRemote && !target.Local
	limiter := r.limiter(remote)
	limiter.Acquire()
	defer limiter.Release()
	return build.Build(r.state, target, remote)
}

// buildAll builds all the targets specified by the given label (which can be :all, but can't be ...).
func (r *runner) buildAll(ctx context.Context, label, dependent core.BuildLabel) error {
	pkg, err := r.Parse(ctx, label, dependent)
	if err != nil {
		return err
	}
	g, gctx := r.group(ctx)
	for _, target := range pkg.AllTargets() {
		if r.state.ShouldInclude(target) {
			g.Go(func() error {
				// N.B. This must go through Build, not buildOne, so we don't build a target twice
				//      if it's reached both via :all and as a dependency of something else.
				_, err := r.Build(gctx, target.Label, dependent)
				return err
			})
		}
	}
	return g.Wait()
}

// Build is the main entrypoint to build a label
func (r *runner) Build(ctx context.Context, label, dependent core.BuildLabel) (*core.BuildTarget, error) {
	if label.IsAllTargets() {
		return r.buildOnce.GetOrSetCtx(ctx, label, func() (*core.BuildTarget, error) {
			return nil, r.buildAll(ctx, label, dependent)
		})
	}
	// N.B. We must parse the target _before_ claiming its entry in buildOnce; parsing its package can
	//      re-enter here for the same label (e.g. a BUILD file that subincludes a target it defines
	//      earlier in the same file) and we'd then deadlock waiting on ourselves.
	target, err := r.parseTarget(ctx, label, dependent)
	if err != nil {
		return nil, err
	}
	return r.buildOnce.GetOrSetCtx(ctx, label, func() (*core.BuildTarget, error) {
		r.progress.numTotal.Add(1)
		defer r.progress.numDone.Add(1)
		return target, r.buildOne(ctx, target)
	})
}

// testOne tests one single target
func (r *runner) testOne(ctx context.Context, target *core.BuildTarget, dependent core.BuildLabel) error {
	if target.IsTest() {
		r.progress.numTotal.Add(int64(r.state.NumTestRuns))
	}
	if _, err := r.Build(ctx, target.Label, dependent); err != nil {
		return err
	}
	if !target.IsTest() {
		return nil
	}
	// Now we're ready to test this target.
	// TODO(peter): Is it okay for none of these to return errors? I _think_ so and we will capture it later?
	remote := r.anyRemote && !target.Local
	limiter := r.limiter(remote)
	if r.state.TestSequentially || r.state.NumTestRuns == 1 { // minor optimisation to avoid creating unnecessary goroutines
		limiter.Acquire()
		defer limiter.Release()
		for run := range int(r.state.NumTestRuns) {
			test.Test(r.state, target, remote, run+1)
			r.progress.numDone.Add(1)
		}
		return nil
	}
	var wg sync.WaitGroup
	for run := range int(r.state.NumTestRuns) {
		wg.Go(func() {
			limiter.Acquire()
			defer limiter.Release()
			test.Test(r.state, target, remote, run+1)
			r.progress.numDone.Add(1)
		})
	}
	wg.Wait()
	return nil
}

// Test is the main entrypoint to run tests for a label
func (r *runner) Test(ctx context.Context, label, dependent core.BuildLabel) error {
	if !label.IsAllTargets() {
		target, err := r.parseTarget(ctx, label, dependent)
		if err != nil {
			return err
		}
		return r.testOne(ctx, target, dependent)
	}
	pkg, err := r.Parse(ctx, label, dependent)
	if err != nil {
		return err
	}
	g, ctx := r.group(ctx)
	for _, target := range pkg.AllTargets() {
		if r.state.ShouldInclude(target) && (target.IsTest() || r.state.NeedCoverage) && !target.AddedPostBuild {
			g.Go(func() error {
				return r.testOne(ctx, target, dependent)
			})
		}
	}
	return g.Wait()
}

// limiter returns either a local or remote limiter that ensures we don't build too many things at once.
func (r *runner) limiter(remote bool) limiter {
	if remote {
		return r.remoteLimiter
	}
	return r.localLimiter
}

func (r *runner) FindOriginalTaskSet(ctx context.Context, targets []core.BuildLabel, needTest, needBuild bool) {
	for _, target := range ReadStdinLabels(targets) {
		r.tasks.Go(func() error {
			return r.findOriginalTask(ctx, target, needTest, needBuild)
		})
	}
}

func (r *runner) queueTargetsForDebug(ctx context.Context, target core.BuildLabel) error {
	if _, err := r.Parse(ctx, target, core.OriginalTarget); err != nil {
		return err
	}
	t := r.state.Graph.TargetOrDie(target)
	for _, tool := range t.AllDebugTools() {
		if l, ok := tool.Label(); ok {
			r.findOriginalTask(ctx, l, false, true)
		}
	}
	for _, data := range t.AllDebugData() {
		if l, ok := data.Label(); ok {
			r.findOriginalTask(ctx, l, false, true)
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

func (r *runner) findOriginalTask(ctx context.Context, target core.BuildLabel, needTest, needBuild bool) error {
	if r.arch != cli.HostArch() {
		target = core.LabelToArch(target, r.arch)
	}
	target = stripHostRepoName(r.state.Config, target)
	if !target.IsAllSubpackages() {
		r.queueTask(ctx, target, needTest, needBuild)
		return nil
	}
	// Any command-line labels with subrepos and ... require us to know where they are in order to
	// walk the directory tree, so we have to make sure the subrepo exists first.
	dir := target.PackageName
	prefix := ""
	if target.Subrepo != "" {
		subrepoLabel := target.SubrepoLabel(r.state)
		if target, err := r.Build(ctx, subrepoLabel, core.OriginalTarget); err != nil {
			return err
		} else if err := r.state.EnsureDownloaded(target); err != nil {
			return err
		}
		// Targets now get activated during parsing, so can be built before we finish parsing their package.
		pkg, err := r.Parse(ctx, subrepoLabel, core.OriginalTarget)
		if err != nil {
			return err
		}
		dir = pkg.Subrepo.Dir(dir)
		prefix = pkg.Subrepo.Dir(prefix)
	}
	for filename := range FindAllBuildFiles(r.state.Config, dir, "") {
		dirname, _ := filepath.Split(filename)
		l := core.NewBuildLabel(strings.TrimLeft(strings.TrimPrefix(strings.TrimRight(dirname, "/"), prefix), "/"), "all")
		l.Subrepo = target.Subrepo
		r.queueTask(ctx, l, needTest, needBuild)
	}
	return nil
}

func (r *runner) queueTask(ctx context.Context, target core.BuildLabel, needTest, needBuild bool) {
	r.state.AddOriginalTarget(target)
	r.tasks.Go(func() error {
		if needTest {
			return r.Test(ctx, target, core.OriginalTarget)
		} else if needBuild {
			_, err := r.Build(ctx, target, core.OriginalTarget)
			// TODO(peter): Ensure this gets downloaded if needed
			return err
		}
		return r.RecursiveParse(ctx, target, core.OriginalTarget)
	})
}

// RegisterPreloads waits for all preloaded subinclude targets to be built, downloads them, and then registers them with
// the interpreter. We have to actually register them otherwise this will return before we build any
// transitive subincludes.
func (r *runner) RegisterPreloads(ctx context.Context, state *core.BuildState, parser *asp.Parser) error {
	g, ctx := r.group(ctx)
	preloads := state.GetPreloadedSubincludes()
	for _, inc := range preloads {
		if inc.IsPseudoTarget() {
			return fmt.Errorf("Can't preload pseudotarget %v", inc)
		}

		// Queue them up asynchronously to feed the queues as quickly as possible
		g.Go(func() error {
			if _, err := r.Build(ctx, inc, core.OriginalTarget); err != nil {
				return err
			}
			return parser.PreloadSubinclude(inc)
		})
	}
	// We must wait for all the subinclude targets to be built otherwise updating the locals might race with parsing
	// a package
	if err := g.Wait(); err != nil {
		return err
	}
	parser.RegisterPreloads(preloads)
	return nil
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

// couldBeArch returns the architecture for a potential subrepo name, if it could be one for
// cross-compiling. Note that this is only a syntactic check; a real subrepo can be named this way too,
// so a caller must satisfy itself that nothing else defines it before treating it as an architecture.
func couldBeArch(name string) (cli.Arch, bool) {
	var arch cli.Arch
	if err := arch.UnmarshalFlag(name); err != nil {
		return arch, false
	}
	return arch, true
}
