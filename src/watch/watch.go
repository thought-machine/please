// +build !bootstrap

// Package watch provides a filesystem watcher that is used to rebuild affected targets.
package watch

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/streamrail/concurrent-map"
	"gopkg.in/op/go-logging.v1"

	"core"
	"build"
	"fs"
)

var log = logging.MustGetLogger("watch")

const debounceInterval = 50 * time.Millisecond

// Watch starts watching the sources of the given labels for changes and triggers
// rebuilds whenever they change.
// It never returns successfully, it will either watch forever or die.
func Watch(state *core.BuildState, labels []core.BuildLabel, run bool, runWatchededBuild func(args []string)) {
	runWorkerScriptIfNecessary(state, labels)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error setting up watcher: %s", err)
	}
	// This sets up the actual watches. It must be done in a separate goroutine.
	files := cmap.New()
	go startWatching(watcher, state, labels, files)
	cmds := commands(state, labels, run)

	cmd := getCommandArgs(state, cmds, labels)

	for {
		select {
		case event := <-watcher.Events:
			log.Info("Event: %s", event)
			if !files.Has(event.Name) {
				log.Notice("Skipping notification for %s", event.Name)
				continue
			}

			// Quick debounce; poll and discard all events for the next brief period.
		outer:
			for {
				select {
				case <-watcher.Events:
				case <-time.After(debounceInterval):
					break outer
				}
			}
			runWatchededBuild(cmd)
		case err := <-watcher.Errors:
			log.Error("Error watching files:", err)
		}
	}
}

func runWorkerScriptIfNecessary(state *core.BuildState, labels []core.BuildLabel) {
	// Check if the build label is a test
	for _, label := range labels {
		target := state.Graph.TargetOrDie(label)
		if worker, _, _ := build.TestWorkerCommand(state, target); target.IsTest && worker != "" {
			build.EnsureWorkerStarted(state, worker, target.Label)
		}
	}
	// Get the worker script, run first

}

func getCommandArgs(state *core.BuildState, commands []string, labels []core.BuildLabel) []string {
	binary, err := os.Executable()
	if err != nil {
		log.Warning("Can't determine current executable, will assume 'plz'")
		binary = "plz"
	}

	cmd := []string {binary}
	cmd = append(cmd, commands...)
	cmd = append(cmd, "-c", state.Config.Build.Config)
	for _, label := range labels {
		cmd = append(cmd, label.String())
	}

	//TODO(luna): Maybe i can find a way to get all that output flags
	if state.ShowAllOutput {
		cmd = append(cmd, "--show_all_output")
	}

	return cmd
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
		for _, datum := range target.Data {
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

// commands returns the plz commands that should be used for the given labels.
func commands(state *core.BuildState, labels []core.BuildLabel, run bool) []string {
	if run {
		if len(labels) > 1 {
			return []string{"run", "parallel"}
		}
		return []string{"run"}
	}
	for _, label := range labels {
		if state.Graph.TargetOrDie(label).IsTest {
			return []string{"test"}
		}
	}
	return []string{"build"}
}
