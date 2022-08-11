package core

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path"
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
		log.Warning("Falling back to Bazel WORKSPACE at %s", path.Join(RepoRoot, "WORKSPACE"))
		usingBazelWorkspace = true
		return RepoRoot
	}
	// Check the config for a default repo location. Of course, we have to load system-level config
	// in order to do that...
	config, err := ReadConfigFiles(defaultGlobalConfigFiles(), nil)
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
		if PathExists(path.Join(dir, filename)) {
			return dir, strings.TrimLeft(initial[len(dir):], "/")
		}
		dir, _ = path.Split(dir)
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

// A SourcePair represents a source file with its source and temporary locations.
// This isn't typically used much by callers; it's just useful to have a single type for channels.
type SourcePair struct{ Src, Tmp string }

// IterSources returns all the sources for a function, allowing for sources that are other rules
// and rules that require transitive dependencies.
// Yielded values are pairs of the original source location and its temporary location for this rule.
// If includeTools is true it yields the target's tools as well.
func IterSources(state *BuildState, graph *BuildGraph, target *BuildTarget, includeTools bool) <-chan SourcePair {
	ch := make(chan SourcePair)
	done := map[string]bool{}
	tmpDir := target.TmpDir()
	go func() {
		for input := range IterInputs(state, graph, target, includeTools, false) {
			fullPaths := input.FullPaths(graph)
			for i, sourcePath := range input.Paths(graph) {
				if tmpPath := path.Join(tmpDir, sourcePath); !done[tmpPath] {
					ch <- SourcePair{fullPaths[i], tmpPath}
					done[tmpPath] = true
				}
			}
		}
		close(ch)
	}()
	return ch
}

// IterInputs iterates all the inputs for a target.
func IterInputs(state *BuildState, graph *BuildGraph, target *BuildTarget, includeTools, sourcesOnly bool) <-chan BuildInput {
	ch := make(chan BuildInput)
	done := map[BuildLabel]bool{}
	var inner func(dependency *BuildTarget)
	inner = func(dependency *BuildTarget) {
		if dependency != target {
			ch <- dependency.Label
		}
		if !state.Config.FeatureFlags.NoIterSourcesMarked {
			// All the sources of this target now count as done
			for _, src := range dependency.AllSources() {
				if label, ok := src.Label(); ok && dependency.IsSourceOnlyDep(label) {
					done[label] = true
				}
			}
		}
		done[dependency.Label] = true
		if target == dependency || (target.NeedsTransitiveDependencies && !dependency.OutputIsComplete) {
			for _, dep := range dependency.BuildDependencies(state) {
				for _, dep2 := range recursivelyProvideFor(graph, target, dependency, dep.Label) {
					if !done[dep2] && !dependency.IsTool(dep2) {
						inner(graph.TargetOrDie(dep2))
					}
				}
			}
		} else {
			for _, dep := range dependency.ExportedDependencies() {
				for _, dep2 := range recursivelyProvideFor(graph, target, dependency, dep) {
					if !done[dep2] {
						inner(graph.TargetOrDie(dep2))
					}
				}
			}
		}
	}
	go func() {
		for _, source := range target.AllSources() {
			recursivelyProvideSource(graph, target, source, ch)
		}
		if includeTools {
			for _, tool := range target.AllTools() {
				recursivelyProvideSource(graph, target, tool, ch)
			}
		}
		if !sourcesOnly {
			inner(target)
		}
		close(ch)
	}()
	return ch
}

// recursivelyProvideFor recursively applies ProvideFor to a target.
func recursivelyProvideFor(graph *BuildGraph, target, dependency *BuildTarget, dep BuildLabel) []BuildLabel {
	depTarget := graph.TargetOrDie(dep)
	ret := depTarget.ProvideFor(dependency)
	if len(ret) == 1 && ret[0] == dep {
		// Dependency doesn't have a require/provide directly on this guy, up to the top-level
		// target. We have to check the dep first to keep things consistent with what targets
		// have actually been built.
		ret = depTarget.ProvideFor(target)
		if len(ret) == 1 && ret[0] == dep {
			return ret
		}
	}
	ret2 := make([]BuildLabel, 0, len(ret))
	for _, r := range ret {
		if r == dep {
			ret2 = append(ret2, r) // Providing itself, don't recurse
		} else {
			ret2 = append(ret2, recursivelyProvideFor(graph, target, dependency, r)...)
		}
	}
	return ret2
}

// recursivelyProvideSource is similar to recursivelyProvideFor but operates on a BuildInput.
func recursivelyProvideSource(graph *BuildGraph, target *BuildTarget, src BuildInput, ch chan BuildInput) {
	if label, ok := src.nonOutputLabel(); ok {
		for _, p := range recursivelyProvideFor(graph, target, target, label) {
			ch <- p
		}
		return
	}
	ch <- src
}

// IterRuntimeFiles yields all the runtime files for a rule (outputs, tools & data files), similar to above.
func IterRuntimeFiles(graph *BuildGraph, target *BuildTarget, absoluteOuts bool, runtimeDir string) <-chan SourcePair {
	done := map[string]bool{}
	ch := make(chan SourcePair)

	pushOut := func(src, out string) {
		if absoluteOuts {
			out = path.Join(RepoRoot, runtimeDir, out)
		}
		if !done[out] {
			ch <- SourcePair{src, out}
			done[out] = true
		}
	}

	go func() {
		outDir := target.OutDir()
		for _, out := range target.Outputs() {
			pushOut(path.Join(outDir, out), out)
		}

		for _, data := range target.AllData() {
			fullPaths := data.FullPaths(graph)
			for i, dataPath := range data.Paths(graph) {
				pushOut(fullPaths[i], dataPath)
			}
		}

		if target.Test != nil {
			for _, tool := range target.AllTestTools() {
				fullPaths := tool.FullPaths(graph)
				for i, toolPath := range tool.Paths(graph) {
					pushOut(fullPaths[i], toolPath)
				}
			}
		}

		if target.Debug != nil {
			for _, data := range target.AllDebugData() {
				fullPaths := data.FullPaths(graph)
				for i, dataPath := range data.Paths(graph) {
					pushOut(fullPaths[i], dataPath)
				}
			}
			for _, tool := range target.AllDebugTools() {
				fullPaths := tool.FullPaths(graph)
				for i, toolPath := range tool.Paths(graph) {
					pushOut(fullPaths[i], toolPath)
				}
			}
		}
		close(ch)
	}()
	return ch
}

// IterInputPaths yields all the transitive input files for a rule (sources & data files), similar to above (again).
func IterInputPaths(graph *BuildGraph, target *BuildTarget) <-chan string {
	// Use a couple of maps to protect us from dep-graph loops and to stop parsing the same target
	// multiple times. We also only want to push files to the channel that it has not already seen.
	donePaths := map[string]bool{}
	doneTargets := map[*BuildTarget]bool{}
	ch := make(chan string)
	var inner func(*BuildTarget)
	inner = func(target *BuildTarget) {
		if !doneTargets[target] {
			// First yield all the sources of the target only ever pushing declared paths to
			// the channel to prevent us outputting any intermediate files.
			for _, source := range target.AllSources() {
				// If the label is nil add any input paths contained here.
				if label, ok := source.nonOutputLabel(); !ok {
					for _, sourcePath := range source.FullPaths(graph) {
						if !donePaths[sourcePath] {
							ch <- sourcePath
							donePaths[sourcePath] = true
						}
					}
					// Otherwise we should recurse for this build label (and gather its sources)
				} else {
					t := graph.TargetOrDie(label)
					for _, d := range recursivelyProvideFor(graph, target, t, t.Label) {
						inner(graph.TargetOrDie(d))
					}
				}
			}

			// Now yield all the data deps of this rule.
			for _, data := range target.AllData() {
				// If the label is nil add any input paths contained here.
				if label, ok := data.Label(); !ok {
					for _, sourcePath := range data.FullPaths(graph) {
						if !donePaths[sourcePath] {
							ch <- sourcePath
							donePaths[sourcePath] = true
						}
					}
					// Otherwise we should recurse for this build label (and gather its sources)
				} else {
					t := graph.TargetOrDie(label)
					for _, d := range recursivelyProvideFor(graph, target, t, t.Label) {
						inner(graph.TargetOrDie(d))
					}
				}
			}

			// Finally recurse for all the deps of this rule.
			for _, dep := range target.Dependencies() {
				for _, d := range recursivelyProvideFor(graph, target, dep, dep.Label) {
					inner(graph.TargetOrDie(d))
				}
			}
			doneTargets[target] = true
		}
	}
	go func() {
		inner(target)
		close(ch)
	}()
	return ch
}

// PrepareSource symlinks a single source file for a build rule.
func PrepareSource(sourcePath string, tmpPath string) error {
	dir := path.Dir(tmpPath)
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

// PrepareSourcePair prepares a source file for a build.
func PrepareSourcePair(pair SourcePair) error {
	if path.IsAbs(pair.Src) {
		return PrepareSource(pair.Src, pair.Tmp)
	}
	return PrepareSource(path.Join(RepoRoot, pair.Src), pair.Tmp)
}

// PrepareRuntimeDir prepares a directory with a target's runtime data for a command to be run on.
func PrepareRuntimeDir(state *BuildState, target *BuildTarget, dir string) error {
	if err := fs.ForceRemove(state.ProcessExecutor, dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		return err
	}

	if err := state.EnsureDownloaded(target); err != nil {
		return err
	}

	for out := range IterRuntimeFiles(state.Graph, target, true, dir) {
		if err := PrepareSourcePair(out); err != nil {
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
			p3 := path.Join(p2, filename)
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
