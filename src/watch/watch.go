// Package watch provides a filesystem watcher that is used to rebuild affected targets.
package watch

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	cmap "github.com/streamrail/concurrent-map"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/run"
)

var log = logging.MustGetLogger("watch")

const debounceInterval = 100 * time.Millisecond

// A CallbackFunc is supplied to Watch in order to trigger a build.
type CallbackFunc func(*core.BuildState, []core.BuildLabel)

// Watch starts watching the sources of the given labels for changes and triggers
// rebuilds whenever they change.
// It never returns successfully, it will either watch forever or die.
func Watch(state *core.BuildState, labels core.BuildLabels, callback CallbackFunc) {
	// This hasn't been set before, do it now.
	state.NeedTests = anyTests(state, labels)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error setting up watcher: %s", err)
	}
	// This sets up the actual watches. It must be done in a separate goroutine.
	files := cmap.New()
	go startWatching(watcher, state, labels, files)

	parentCtx, cancelParent := context.WithCancel(context.Background())
	cli.AtExit(func() {
		cancelParent()
		time.Sleep(5 * time.Millisecond) // Brief pause to give the cancel() call time to progress before the process dies
	})

	ctx, cancel := context.WithCancel(parentCtx)

	// The initial setup only builds targets, it doesn't test or run things.
	// Do one of those now if requested.
	if state.NeedTests || state.NeedRun {
		build(ctx, state, labels, callback)
	}

	for {
		select {
		case event := <-watcher.Events:
			log.Info("Event: %s", event)
			if !files.Has(event.Name) {
				log.Notice("Skipping notification for %s", event.Name)
				continue
			}
			// Kill any previous process.
			cancel()
			ctx, cancel = context.WithCancel(parentCtx)

			// Quick debounce; poll and discard all events for the next brief period.
		outer:
			for {
				select {
				case <-watcher.Events:
				case <-time.After(debounceInterval):
					break outer
				}
			}
			build(ctx, state, labels, callback)
		case err := <-watcher.Errors:
			log.Error("Error watching files:", err)
		}
	}
}

func startWatching(watcher *fsnotify.Watcher, state *core.BuildState, labels []core.BuildLabel, files cmap.ConcurrentMap) {
	// Deduplicate seen targets & sources.
	targets := map[*core.BuildTarget]struct{}{}
	dirs := map[string]struct{}{}

	var startWatch func(*core.BuildTarget)
	startWatch = func(target *core.BuildTarget) {
		if _, present := targets[target]; present {
			return
		}
		targets[target] = struct{}{}
		for _, source := range target.AllSources() {
			addSource(watcher, state, source, dirs, files)
		}
		for _, datum := range target.AllData() {
			addSource(watcher, state, datum, dirs, files)
		}
		for _, dep := range target.Dependencies() {
			startWatch(dep)
		}
		pkg := state.Graph.PackageOrDie(target.Label)
		if !files.Has(pkg.Filename) {
			log.Notice("Adding watch on %s", pkg.Filename)
			files.Set(pkg.Filename, struct{}{})
		}
		for _, subinclude := range pkg.Subincludes {
			startWatch(state.Graph.TargetOrDie(subinclude))
		}
	}

	for _, label := range labels {
		startWatch(state.Graph.TargetOrDie(label))
	}
	// Drop a message here so they know when it's actually ready to go.
	fmt.Println("And now my watch begins...")
}

func addSource(watcher *fsnotify.Watcher, state *core.BuildState, source core.BuildInput, dirs map[string]struct{}, files cmap.ConcurrentMap) {
	if source.Label() == nil {
		for _, src := range source.Paths(state.Graph) {
			if err := fs.Walk(src, func(src string, isDir bool) error {
				files.Set(src, struct{}{})
				if !path.IsAbs(src) {
					files.Set("./"+src, struct{}{})
				}
				dir := src
				if !isDir {
					dir = path.Dir(src)
				}
				if _, present := dirs[dir]; !present {
					log.Notice("Adding watch on %s", dir)
					dirs[dir] = struct{}{}
					if err := watcher.Add(dir); err != nil {
						log.Error("Failed to add watch on %s: %s", src, err)
					}
				}
				return nil
			}); err != nil {
				log.Error("Failed to add watch on %s: %s", src, err)
			}
		}
	}
}

// anyTests returns true if any of the given labels refer to tests.
func anyTests(state *core.BuildState, labels []core.BuildLabel) bool {
	for _, l := range labels {
		if state.Graph.TargetOrDie(l).IsTest {
			return true
		}
	}
	return false
}

// build invokes a single build while watching.
func build(ctx context.Context, state *core.BuildState, labels []core.BuildLabel, callback CallbackFunc) {
	// Set up a new state & copy relevant parts off the existing one.
	ns := core.NewBuildState(state.Config)
	ns.Cache = state.Cache
	ns.VerifyHashes = state.VerifyHashes
	ns.NumTestRuns = state.NumTestRuns
	ns.NeedTests = state.NeedTests
	ns.NeedRun = state.NeedRun
	ns.Watch = true
	ns.CleanWorkdirs = state.CleanWorkdirs
	ns.DebugTests = state.DebugTests
	ns.ShowAllOutput = state.ShowAllOutput
	ns.StartTime = time.Now()
	callback(ns, labels)
	if state.NeedRun {
		// Don't wait for this, its lifetime will be controlled by the context.
		als := make([]core.AnnotatedOutputLabel, len(labels))
		for i, l := range labels {
			als[i] = core.AnnotatedOutputLabel{
				BuildLabel: l,
			}
		}
		go run.Parallel(ctx, state, als, nil, state.Config.Please.NumThreads, run.Default, false, false, false, false, "")
	}
}
