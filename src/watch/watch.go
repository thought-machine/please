// +build watch

// Package watch provides a filesystem watcher that is used to rebuild affected targets.
package watch

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kardianos/osext"
	"github.com/streamrail/concurrent-map"
	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("watch")

const debounceInterval = 50 * time.Millisecond

// Watch starts watching the sources of the given labels for changes and triggers
// rebuilds whenever they change.
// It never returns successfully, it will either watch forever or die.
func Watch(state *core.BuildState, labels []core.BuildLabel) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error setting up watcher: %s", err)
	}
	// This sets up the actual watches. It must be done in a separate goroutine.
	files := cmap.New()
	go startWatching(watcher, state, labels, files)

	// If any of the targets are tests, we'll run plz test, otherwise just plz build.
	command := "build"
	for _, label := range labels {
		if state.Graph.TargetOrDie(label).IsTest {
			command = "test"
			break
		}
	}
	log.Notice("Command: %s", command)

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
			runBuild(state, command, labels)
		case err := <-watcher.Errors:
			log.Error("Error watching files:", err)
		}
	}
}

func runBuild(state *core.BuildState, command string, labels []core.BuildLabel) {
	binary, err := osext.Executable()
	if err != nil {
		log.Warning("Can't determine current executable, will assume 'plz'")
		binary = "plz"
	}
	cmd := exec.Command(binary, command)
	cmd.Args = append(cmd.Args, "-c", state.Config.Build.Config)
	for _, label := range labels {
		cmd.Args = append(cmd.Args, label.String())
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Notice("Running %s %s...", binary, command)
	if err := cmd.Run(); err != nil {
		// Only log the error if it's not a straightforward non-zero exit; the user will presumably
		// already have been pestered about that.
		if _, ok := err.(*exec.ExitError); !ok {
			log.Error("Failed to run %s: %s", binary, err)
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
		for _, datum := range target.Data {
			addSource(watcher, state, datum, dirs, files)
		}
		for _, dep := range target.Dependencies() {
			startWatch(dep)
		}
		pkg := state.Graph.PackageOrDie(target.Label.PackageName)
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
			if err := filepath.Walk(src, func(src string, info os.FileInfo, err error) error {
				files.Set(src, struct{}{})
				dir := src
				if info, err := os.Stat(src); err == nil && !info.IsDir() {
					dir = path.Dir(src)
				}
				if _, present := dirs[dir]; !present {
					log.Notice("Adding watch on %s", dir)
					dirs[dir] = struct{}{}
					if err := watcher.Add(dir); err != nil {
						log.Error("Failed to add watch on %s: %s", src, err)
					}
				}
				return err
			}); err != nil {
				log.Error("Failed to add watch on %s: %s", src, err)
			}
		}
	}
}
