package core

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Root of the repository
var RepoRoot string

// Initial working directory
var initialWorkingDir string

// Initial subdir of the working directory, ie. what package did we start in.
var initialPackage string

const DirPermissions = os.ModeDir | 0775

// FindRepoRoot returns the root directory of the current repo and sets the initial working dir.
// Dies on failure if 'die' is set.
func FindRepoRoot(die bool) {
	initialWorkingDir, _ = os.Getwd()
	RepoRoot, initialPackage = getRepoRoot(die)
}

// InitialPackage returns a label corresponding to the initial package we started in.
func InitialPackage() []BuildLabel {
	// It's possible to start off in directories that aren't legal package names, because
	// our package naming is stricter than directory naming requirements.
	// In that case move up until we find somewhere we can run from.
	dir := initialPackage
	for dir != "." {
		if label, err := TryNewBuildLabel(dir, "..."); err == nil {
			return []BuildLabel{label}
		}
		dir = filepath.Dir(dir)
	}
	return WholeGraph
}

// getRepoRoot returns the root directory of the current repo and the initial package.
func getRepoRoot(die bool) (string, string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't determine working directory: %s", err)
	}
	// Walk up directories looking for a .plzconfig file, which we use to identify the root.
	initial := dir
	for dir != "" {
		if PathExists(path.Join(dir, ConfigFileName)) {
			return dir, strings.TrimLeft(initial[len(dir):], "/")
		}
		dir, _ = path.Split(dir)
		dir = strings.TrimRight(dir, "/")
	}
	if die {
		log.Fatalf("Couldn't locate the repo root. Are you sure you're inside a plz repo?")
	}
	return "", ""
}

// Returns true if the build was initiated from the repo root.
// Used to provide slightly nicer output in some places.
func StartedAtRepoRoot() bool {
	return RepoRoot == initialWorkingDir
}

func PathExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

// CopyFile copies a file from 'from' to 'to', with an attempt to perform a copy & rename
// to avoid chaos if anything goes wrong partway.
func CopyFile(from string, to string, mode os.FileMode) error {
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromFile.Close()
	return WriteFile(fromFile, to, mode)
}

// WriteFile writes data from a reader to the file named 'to', with an attempt to perform
// a copy & rename to avoid chaos if anything goes wrong partway.
func WriteFile(fromFile io.Reader, to string, mode os.FileMode) error {
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	dir, file := path.Split(to)
	if err := os.MkdirAll(dir, DirPermissions); err != nil {
		return err
	}
	tempFile, err := ioutil.TempFile(dir, file)
	if err != nil {
		return err
	}
	if _, err := io.Copy(tempFile, fromFile); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	// OK, now file is written; adjust permissions appropriately.
	if mode == 0 {
		mode = 0664
	}
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		return err
	}
	// And move it to its final destination.
	return os.Rename(tempFile.Name(), to)
}

// Copies either a single file or a directory.
// If 'link' is true then we'll attempt to hardlink files instead of copying them
// (on failure we still attempt a copy).
func RecursiveCopyFile(from string, to string, mode os.FileMode, link bool) error {
	if info, err := os.Stat(from); err == nil && info.IsDir() {
		return filepath.Walk(from, func(name string, info os.FileInfo, err error) error {
			dest := path.Join(to, name[len(from):])
			if err != nil {
				return err
			} else if info.IsDir() {
				return os.MkdirAll(dest, DirPermissions)
			} else if (info.Mode() & os.ModeSymlink) != 0 {
				fi, err := os.Stat(name)
				if err != nil {
					return err
				}
				if fi.IsDir() {
					return RecursiveCopyFile(name+"/", dest+"/", mode, link)
				} else {
					return copyOrLinkFile(name, dest, mode, link)
				}
			} else {
				return copyOrLinkFile(name, dest, mode, link)
			}
		})
	} else {
		return copyOrLinkFile(from, to, mode, link)
	}
}

// Either copies or hardlinks a file based on the link argument.
func copyOrLinkFile(from, to string, mode os.FileMode, link bool) error {
	if link {
		if err := os.Link(from, to); err == nil {
			return nil
		}
	}
	return CopyFile(from, to, mode)
}

// TimeoutError is the type of error that's issued when a command times out.
type TimeoutError int

// Error formats this error as a string.
func (i TimeoutError) Error() string {
	return fmt.Sprintf("Timeout (%d seconds) exceeded", i)
}

// ExecWithTimeout runs an external command with a timeout.
// If the command times out the returned error will be a TimeoutError.
func ExecWithTimeout(cmd *exec.Cmd, timeout int, defaultTimeout int) ([]byte, error) {
	var out bytes.Buffer
	if timeout == 0 {
		timeout = defaultTimeout
	}
	cmd.Stdout = &out
	cmd.Stderr = &out
	ch := make(chan error)
	go func() { ch <- cmd.Run() }()
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			log.Errorf("Process %d could not be killed after exceeding timeout of %d seconds", cmd.Process.Pid, timeout)
		}
		return out.Bytes(), TimeoutError(timeout)
	case err := <-ch:
		return out.Bytes(), err
	}
}

type sourcePair struct{ Src, Tmp string }

// IterSources returns all the sources for a function, allowing for sources that are other rules
// and rules that require transitive dependencies.
// Yielded values are pairs of the original source location and its temporary location for this rule.
func IterSources(graph *BuildGraph, target *BuildTarget) <-chan sourcePair {
	ch := make(chan sourcePair)
	done := map[BuildLabel]bool{}
	donePaths := map[string]bool{}
	tmpDir := target.TmpDir()
	var inner func(dependency *BuildTarget)
	inner = func(dependency *BuildTarget) {
		if target == dependency {
			// This is the current build rule, so link its sources.
			for _, source := range dependency.AllSources() {
				fullPaths := source.FullPaths(graph)
				for i, sourcePath := range source.Paths(graph) {
					tmpPath := path.Join(tmpDir, sourcePath)
					ch <- sourcePair{fullPaths[i], tmpPath}
					donePaths[tmpPath] = true
				}
			}
		} else {
			// This is a dependency of the rule, so link its outputs.
			for _, dep := range dependency.Outputs() {
				depPath := path.Join(dependency.OutDir(), dep)
				tmpPath := path.Join(tmpDir, dependency.Label.PackageName, dep)
				if !donePaths[tmpPath] {
					ch <- sourcePair{depPath, tmpPath}
					donePaths[tmpPath] = true
				}
			}
			// Mark any label-type outputs as done.
			for _, out := range dependency.DeclaredOutputs() {
				if LooksLikeABuildLabel(out) {
					label := ParseBuildLabel(out, target.Label.PackageName)
					done[label] = true
				}
			}
		}
		done[dependency.Label] = true
		if target == dependency || (target.NeedsTransitiveDependencies && !dependency.OutputIsComplete) {
			for _, dep := range dependency.Dependencies() {
				if !done[dep.Label] && !dependency.IsTool(dep.Label) {
					inner(dep)
				}
			}
		} else {
			for _, dep := range dependency.ExportedDependencies() {
				for _, dep2 := range recursivelyProvideFor(graph, target, dep) {
					if !done[dep2] {
						inner(graph.TargetOrDie(dep2))
					}
				}
			}
		}
	}
	go func() {
		inner(target)
		close(ch)
	}()
	return ch
}

// recursivelyProvideFor recursively applies ProvideFor to a target.
func recursivelyProvideFor(graph *BuildGraph, target *BuildTarget, dep BuildLabel) []BuildLabel {
	ret := graph.TargetOrDie(dep).ProvideFor(target)
	if len(ret) == 1 && ret[0] == dep {
		return ret // Providing itself, don't recurse
	}
	ret2 := []BuildLabel{}
	for _, r := range ret {
		ret2 = append(ret2, recursivelyProvideFor(graph, target, r)...)
	}
	return ret2
}

// Yields all the runtime files for a rule (outputs & data files), similar to above.
func IterRuntimeFiles(graph *BuildGraph, target *BuildTarget, absoluteOuts bool) <-chan sourcePair {
	done := map[string]bool{}
	ch := make(chan sourcePair)

	makeOut := func(out string) string {
		if absoluteOuts {
			return path.Join(RepoRoot, target.TestDir(), out)
		} else {
			return out
		}
	}

	pushOut := func(src, out string) {
		out = makeOut(out)
		if !done[out] {
			ch <- sourcePair{src, out}
			done[out] = true
		}
	}

	var inner func(*BuildTarget)
	inner = func(target *BuildTarget) {
		for _, out := range target.Outputs() {
			pushOut(path.Join(target.OutDir(), out), out)
		}
		for _, data := range target.Data {
			fullPaths := data.FullPaths(graph)
			for i, dataPath := range data.Paths(graph) {
				pushOut(fullPaths[i], dataPath)
			}
			if label := data.Label(); label != nil {
				for _, dep := range graph.TargetOrDie(*label).ExportedDependencies() {
					inner(graph.TargetOrDie(dep))
				}
			}
		}
		for _, dep := range target.ExportedDependencies() {
			inner(graph.TargetOrDie(dep))
		}
	}
	go func() {
		inner(target)
		close(ch)
	}()
	return ch
}

// Yields all the transitive input files for a rule (sources & data files), similar to above (again).
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
				if label := source.Label(); label == nil {
					for _, sourcePath := range source.FullPaths(graph) {
						if !donePaths[sourcePath] {
							ch <- sourcePath
							donePaths[sourcePath] = true
						}
					}
					// Otherwise we should recurse for this build label (and gather its sources)
				} else {
					inner(graph.TargetOrDie(*label))
				}
			}

			// Now yield all the data deps of this rule.
			for _, data := range target.Data {
				// If the label is nil add any input paths contained here.
				if label := data.Label(); label == nil {
					for _, sourcePath := range data.FullPaths(graph) {
						if !donePaths[sourcePath] {
							ch <- sourcePath
							donePaths[sourcePath] = true
						}
					}
					// Otherwise we should recurse for this build label (and gather its sources)
				} else {
					inner(graph.TargetOrDie(*label))
				}
			}

			// Finally recurse for all the deps of this rule.
			for _, dep := range target.Dependencies() {
				inner(dep)
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

// Symlinks a single source file for a build rule.
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
	return RecursiveCopyFile(sourcePath, tmpPath, 0, true)
}

func PrepareSourcePair(pair sourcePair) error {
	if path.IsAbs(pair.Src) {
		return PrepareSource(pair.Src, pair.Tmp)
	}
	return PrepareSource(path.Join(RepoRoot, pair.Src), pair.Tmp)
}

func PostBuildOutputFileName(target *BuildTarget) string {
	return ".build_output_" + target.Label.Name
}

// CollapseHash combines our usual four-part hash into one by XOR'ing them together.
// This helps keep things short in places where sometimes we get complaints about filenames being too long (?)
// and where we don't especially care about breaking out the individual parts of hashes, which
// is important for many parts of the system.
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
