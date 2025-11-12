package core

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/fs"
)

// RepoRoot is the root of the Please repository
var RepoRoot string

// InitialWorkingDir is the directory we began in. Early on we chdir() to the repo root but for
// some things we need to remember this.
var InitialWorkingDir string

// InitialPackagePath is the initial subdir of the working directory, ie. what package did we start in.
// This is similar but not identical to InitialWorkingDir.
var InitialPackagePath string

// usingBazelWorkspace is true if we detected a Bazel WORKSPACE file to find our repo root.
var usingBazelWorkspace bool

// DirPermissions are the default permission bits we apply to directories.
const DirPermissions = os.ModeDir | 0775

// FindRepoRoot returns the root directory of the current repo and sets the initial working dir.
// It returns true if the repo root was found.
func FindRepoRoot() bool {
	InitialWorkingDir, _ = os.Getwd()
	RepoRoot, InitialPackagePath = getRepoRoot(ConfigFileName)
	return RepoRoot != ""
}

// MustFindRepoRoot returns the root directory of the current repo and sets the initial working dir.
// It dies on failure, although will fall back to looking for a Bazel WORKSPACE file first.
func MustFindRepoRoot() string {
	if RepoRoot != "" {
		return RepoRoot
	} else if FindRepoRoot() {
		return RepoRoot
	}
	RepoRoot, InitialPackagePath = getRepoRoot("WORKSPACE")
	if RepoRoot != "" {
		log.Warning("No .plzconfig file found to define the repo root.")
		log.Warning("Falling back to Bazel WORKSPACE at %s", filepath.Join(RepoRoot, "WORKSPACE"))
		usingBazelWorkspace = true
		return RepoRoot
	}
	// Check the config for a default repo location. Of course, we have to load system-level config
	// in order to do that...
	config, err := ReadConfigFiles(fs.HostFS, defaultGlobalConfigFiles(), nil)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}
	if config.Please.DefaultRepo != "" {
		log.Warning("Using default repo at %s", config.Please.DefaultRepo)
		RepoRoot = fs.ExpandHomePath(config.Please.DefaultRepo)
		return RepoRoot
	}
	log.Fatalf("Couldn't locate the repo root. Are you sure you're inside a plz repo?")
	return ""
}

// InitialPackage returns a label corresponding to the initial package we started in.
func InitialPackage() []BuildLabel {
	// It's possible to start off in directories that aren't legal package names, because
	// our package naming is stricter than directory naming requirements.
	// In that case move up until we find somewhere we can run from.
	dir := InitialPackagePath
	for dir != "." {
		if label, err := TryNewBuildLabel(dir, "test"); err == nil {
			label.Name = "..."
			return []BuildLabel{label}
		}
		dir = filepath.Dir(dir)
	}
	return WholeGraph
}

// getRepoRoot returns the root directory of the current repo and the initial package.
func getRepoRoot(filename string) (string, string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't determine working directory: %s", err)
	}
	// Walk up directories looking for a .plzconfig file, which we use to identify the root.
	initial := dir
	for dir != "" {
		if PathExists(filepath.Join(dir, filename)) {
			return dir, strings.TrimLeft(initial[len(dir):], "/")
		}
		dir, _ = filepath.Split(dir)
		dir = strings.TrimRight(dir, "/")
	}
	return "", ""
}

// StartedAtRepoRoot returns true if the build was initiated from the repo root.
// Used to provide slightly nicer output in some places.
func StartedAtRepoRoot() bool {
	return RepoRoot == InitialWorkingDir
}

// ReturnToInitialWorkingDir changes directory back to where plz was first started from.
func ReturnToInitialWorkingDir() {
	if err := os.Chdir(InitialWorkingDir); err != nil {
		log.Error("Failed to change directory to %s: %s", InitialWorkingDir, err)
	}
}

// IterSources returns all the sources for a function, allowing for sources that are other rules
// and rules that require transitive dependencies.
// Yielded values are pairs of the original source location and its temporary location for this rule.
// If includeTools is true it yields the target's tools as well.
func IterSources(state *BuildState, graph *BuildGraph, target *BuildTarget, includeTools bool) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		done := map[string]bool{}
		tmpDir := target.TmpDir()
		for input := range IterInputs(state, graph, target, includeTools, false) {
			fullPaths := input.FullPaths(graph)
			for i, sourcePath := range input.Paths(graph) {
				if tmpPath := filepath.Join(tmpDir, sourcePath); !done[tmpPath] {
					if !yield(fullPaths[i], tmpPath) {
						return
					}
					done[tmpPath] = true
				}
			}
		}
	}
}

// IterInputs iterates all the inputs for a target.
func IterInputs(state *BuildState, graph *BuildGraph, target *BuildTarget, includeTools, sourcesOnly bool) iter.Seq[BuildInput] {
	return func(yield func(BuildInput) bool) {
		done := map[BuildLabel]bool{}
		recursivelyProvideSource := func(target *BuildTarget, src BuildInput) bool {
			if label, ok := src.nonOutputLabel(); ok {
				for p := range recursivelyProvideFor(graph, target, target, label) {
					if !yield(p) {
						return false
					}
					for runDep := range graph.TargetOrDie(p).IterAllRuntimeDependencies(graph) {
						log.Debug("core.IterInputs: recursivelyProvideSource: %s", runDep.String())
						if !yield(runDep) {
							return false
						}
					}
				}
				return true
			}
			return yield(src)
		}
		var inner func(dependency *BuildTarget) bool
		inner = func(dependency *BuildTarget) bool {
			if dependency != target {
				if !yield(dependency.Label) {
					return false
				}
				for runDep := range graph.TargetOrDie(dependency.Label).IterAllRuntimeDependencies(graph) {
					log.Debug("core.IterInputs: inner: %s", runDep.String())
					if !yield(runDep) {
						return false
					}
				}
			}

			done[dependency.Label] = true
			if target == dependency || (target.NeedsTransitiveDependencies && !dependency.OutputIsComplete) {
				for _, dep := range dependency.BuildDependencies() {
					for dep2 := range recursivelyProvideFor(graph, target, dependency, dep.Label) {
						if !done[dep2] && !dependency.IsTool(dep2) {
							if !inner(graph.TargetOrDie(dep2)) {
								return false
							}
						}
					}
				}
			} else {
				for _, dep := range dependency.ExportedDependencies() {
					for dep2 := range recursivelyProvideFor(graph, target, dependency, dep) {
						if !done[dep2] {
							if !inner(graph.TargetOrDie(dep2)) {
								return false
							}
						}
					}
				}
			}
			return true
		}
		for _, source := range target.AllSources() {
			if !recursivelyProvideSource(target, source) {
				return
			}
		}
		if includeTools {
			for _, tool := range target.AllTools() {
				if !recursivelyProvideSource(target, tool) {
					return
				}
			}
		}
		if !sourcesOnly {
			inner(target)
		}
	}
}

// recursivelyProvideFor recursively applies ProvideFor to a target.
func recursivelyProvideFor(graph *BuildGraph, target, dependency *BuildTarget, dep BuildLabel) iter.Seq[BuildLabel] {
	return func(yield func(BuildLabel) bool) {
		depTarget := graph.TargetOrDie(dep)
		ret := depTarget.ProvideFor(dependency)
		if len(ret) == 1 && ret[0] == dep {
			// Dependency doesn't have a require/provide directly on this guy, up to the top-level
			// target. We have to check the dep first to keep things consistent with what targets
			// have actually been built.
			ret = depTarget.ProvideFor(target)
			if len(ret) == 1 && ret[0] == dep {
				yield(ret[0])
				return
			}
		}
		for _, r := range ret {
			if r == dep {
				if !yield(r) { // Providing itself, don't recurse
					return
				}
			} else {
				for p := range recursivelyProvideFor(graph, target, dependency, r) {
					if !yield(p) {
						return
					}
				}
			}
		}
	}
}

// IterRuntimeFiles yields all the runtime files for a rule (outputs, tools & data files), similar to above.
func IterRuntimeFiles(graph *BuildGraph, target *BuildTarget, absoluteOuts bool, runtimeDir string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		done := map[string]bool{}

		pushOut := func(src, out string) bool {
			if absoluteOuts {
				out = filepath.Join(RepoRoot, runtimeDir, out)
			}
			if !done[out] {
				done[out] = true
				if !yield(src, out) {
					return false
				}
			}
			return true
		}

		outDir := target.OutDir()
		for _, out := range target.Outputs() {
			if !pushOut(filepath.Join(outDir, out), out) {
				return
			}
		}

		for runDep := range target.IterAllRuntimeDependencies(graph) {
			log.Debug("core.IterRuntimeFiles: %s", runDep.String())
			fullPaths := runDep.FullPaths(graph)
			for i, depPath := range runDep.Paths(graph) {
				if !pushOut(fullPaths[i], depPath) {
					return
				}
			}
		}

		for _, data := range target.AllData() {
			fullPaths := data.FullPaths(graph)
			for i, dataPath := range data.Paths(graph) {
				if !pushOut(fullPaths[i], dataPath) {
					return
				}
			}
			label, ok := data.Label()
			if !ok {
				continue
			}
			for runDep := range graph.TargetOrDie(label).IterAllRuntimeDependencies(graph) {
				log.Debug("core.IterRuntimeFiles: data: %s", runDep.String())
				fullPaths := runDep.FullPaths(graph)
				for i, depPath := range runDep.Paths(graph) {
					if !pushOut(fullPaths[i], depPath) {
						return
					}
				}
			}
		}

		if target.Test != nil {
			for _, tool := range target.AllTestTools() {
				fullPaths := tool.FullPaths(graph)
				for i, toolPath := range tool.Paths(graph) {
					if !pushOut(fullPaths[i], toolPath) {
						return
					}
				}
				label, ok := tool.Label()
				if !ok {
					continue
				}
				for runDep := range graph.TargetOrDie(label).IterAllRuntimeDependencies(graph) {
					log.Debug("core.IterRuntimeFiles: test_tools: %s", runDep.String())
					fullPaths := runDep.FullPaths(graph)
					for i, depPath := range runDep.Paths(graph) {
						if !pushOut(fullPaths[i], depPath) {
							return
						}
					}
				}
			}
		}

		if target.Debug != nil {
			for _, data := range target.AllDebugData() {
				fullPaths := data.FullPaths(graph)
				for i, dataPath := range data.Paths(graph) {
					if !pushOut(fullPaths[i], dataPath) {
						return
					}
				}
				label, ok := data.Label()
				if !ok {
					continue
				}
				for runDep := range graph.TargetOrDie(label).IterAllRuntimeDependencies(graph) {
					log.Debug("core.IterRuntimeFiles: debug_data: %s", runDep.String())
					fullPaths := runDep.FullPaths(graph)
					for i, depPath := range runDep.Paths(graph) {
						if !pushOut(fullPaths[i], depPath) {
							return
						}
					}
				}
			}
			for _, tool := range target.AllDebugTools() {
				fullPaths := tool.FullPaths(graph)
				for i, toolPath := range tool.Paths(graph) {
					if !pushOut(fullPaths[i], toolPath) {
						return
					}
				}
				label, ok := tool.Label()
				if !ok {
					continue
				}
				for runDep := range graph.TargetOrDie(label).IterAllRuntimeDependencies(graph) {
					log.Debug("core.IterRuntimeFiles: debug_tools: %s", runDep.String())
					fullPaths := runDep.FullPaths(graph)
					for i, depPath := range runDep.Paths(graph) {
						if !pushOut(fullPaths[i], depPath) {
							return
						}
					}
				}
			}
		}
	}
}

// IterInputPaths yields all the transitive input files for a rule (sources & data files), similar to above (again).
func IterInputPaths(graph *BuildGraph, target *BuildTarget) iter.Seq[string] {
	return func(yield func(string) bool) {
		// Use a couple of maps to protect us from dep-graph loops and to stop parsing the same target
		// multiple times. We also only want to push files to the channel that it has not already seen.
		donePaths := map[string]bool{}
		doneTargets := map[*BuildTarget]bool{}
		var inner func(*BuildTarget) bool
		inner = func(target *BuildTarget) bool {
			if !doneTargets[target] {
				// First yield all the sources of the target only ever pushing declared paths to
				// the channel to prevent us outputting any intermediate files.
				for _, source := range target.AllSources() {
					// If the label is nil add any input paths contained here.
					if label, ok := source.nonOutputLabel(); !ok {
						// Don't emit inputs from subrepos; they appear to be files in many ways but they are themselves generated
						if _, ok := source.(SubrepoFileLabel); ok {
							continue
						}
						for _, sourcePath := range source.FullPaths(graph) {
							if !donePaths[sourcePath] {
								if !yield(sourcePath) {
									return false
								}
								donePaths[sourcePath] = true
							}
						}
						// Otherwise we should recurse for this build label (and gather its sources)
					} else {
						t := graph.TargetOrDie(label)
						for d := range recursivelyProvideFor(graph, target, t, t.Label) {
							if !inner(graph.TargetOrDie(d)) {
								return false
							}
						}
					}
				}

				// Now yield all the data deps of this rule.
				for _, data := range target.AllData() {
					// If the label is nil add any input paths contained here.
					if label, ok := data.Label(); !ok {
						// Don't emit inputs from subrepos; they appear to be files in many ways but they are themselves generated
						if _, ok := data.(SubrepoFileLabel); ok {
							continue
						}
						for _, sourcePath := range data.FullPaths(graph) {
							if !donePaths[sourcePath] {
								if !yield(sourcePath) {
									return false
								}
								donePaths[sourcePath] = true
							}
						}
						// Otherwise we should recurse for this build label (and gather its sources)
					} else {
						t := graph.TargetOrDie(label)
						for d := range recursivelyProvideFor(graph, target, t, t.Label) {
							if !inner(graph.TargetOrDie(d)) {
								return false
							}
						}
					}
				}

				// Finally recurse for all the deps of this rule.
				for _, dep := range target.Dependencies() {
					for d := range recursivelyProvideFor(graph, target, dep, dep.Label) {
						if !inner(graph.TargetOrDie(d)) {
							return false
						}
					}
				}
				doneTargets[target] = true
			}
			return true
		}
		inner(target)
	}
}

// PrepareSource symlinks a single source file for a build rule.
func PrepareSource(sourcePath string, tmpPath string) error {
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(RepoRoot, sourcePath)
	}
	dir := filepath.Dir(tmpPath)
	if !PathExists(dir) {
		if err := os.MkdirAll(dir, DirPermissions); err != nil {
			return err
		}
	}
	if !PathExists(sourcePath) {
		return fmt.Errorf("Source file %s doesn't exist", sourcePath)
	}
	return fs.RecursiveLink(sourcePath, tmpPath)
}

// PrepareRuntimeDir prepares a directory with a target's runtime data for a command to be run on.
func PrepareRuntimeDir(state *BuildState, target *BuildTarget, dir string) error {
	if err := fs.RemoveAll(dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		return err
	}

	if err := state.EnsureDownloaded(target); err != nil {
		return err
	}

	for src, tmp := range IterRuntimeFiles(state.Graph, target, true, dir) {
		if err := PrepareSource(src, tmp); err != nil {
			return err
		}
	}

	return nil
}

// CollapseHash combines our usual four-part hash into one by XOR'ing them together.
// This helps keep things short in places where sometimes we get complaints about filenames being
// too long (this is most noticeable on e.g. Ubuntu with an encrypted home directory, but
// not an entire encrypted disk) and where we don't especially care about breaking out the
// individual parts of hashes, which is important for many parts of the system.
func CollapseHash(key []byte) []byte {
	short := [sha1.Size]byte{}
	// We store the rule hash twice, if it's repeated we must make sure not to xor it
	// against itself.
	if bytes.Equal(key[0:sha1.Size], key[sha1.Size:2*sha1.Size]) {
		for i := 0; i < sha1.Size; i++ {
			short[i] = key[i] ^ key[i+2*sha1.Size] ^ key[i+3*sha1.Size]
		}
	} else {
		for i := 0; i < sha1.Size; i++ {
			short[i] = key[i] ^ key[i+sha1.Size] ^ key[i+2*sha1.Size] ^ key[i+3*sha1.Size]
		}
	}
	return short[:]
}

// LookPath does roughly the same as exec.LookPath, i.e. looks for the named file on the path.
// The main difference is that it looks based on our config which isn't necessarily the same
// as the external environment variable.
func LookPath(filename string, paths []string) (string, error) {
	for _, p := range paths {
		for _, p2 := range strings.Split(p, ":") {
			p3 := filepath.Join(p2, filename)
			if _, err := os.Stat(p3); err == nil {
				return p3, nil
			}
		}
	}
	return "", fmt.Errorf("%s not found in path %s", filename, strings.Join(paths, ":"))
}

// LookBuildPath is like LookPath but takes the config's build path into account.
func LookBuildPath(filename string, config *Configuration) (string, error) {
	return LookPath(filename, config.Path())
}

// PathExists is an alias to fs.PathExists.
// TODO(peterebden): Remove and migrate everything over.
func PathExists(filename string) bool {
	return fs.PathExists(filename)
}
