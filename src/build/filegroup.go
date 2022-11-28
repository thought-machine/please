// Logic relating to building filegroups.
//
// Unlike most targets, filegroups are special in that (1) they are known to the
// system and have a custom implementation and (2) multiple filegroups can output
// the same file. This does lead to a potential race condition where we have to
// be sure to build each output file only once.
// Currently this is implemented by a single thread that builds them all; there
// are other schemes we could have but this is simple enough (and since we link
// them rather than copying there should not be a lot of I/O wait).

package build

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// Used to ensure we only write our dummy go.mod once.
var goModOnce sync.Once

// Init initialises common resources for the build package.
func Init(state *core.BuildState) {
	theFilegroupBuilder = &filegroupBuilder{
		built: map[string]bool{},
	}
	state.TargetHasher = newTargetHasher(state)
}

// A filegroupBuilder is a singleton that we have that builds all filegroups.
// This works around the problem where multiple filegroups can output the same
// file, which means that if built simultaneously they can fight with one another.
type filegroupBuilder struct {
	mutex sync.Mutex
	built map[string]bool
}

var theFilegroupBuilder *filegroupBuilder

func isSameFileContent(state *core.BuildState, hashTimestamp bool, from, to string) (bool, error) {
	if !fs.PathExists(to) {
		return false, nil
	}
	if from == to || fs.IsSameFile(from, to) {
		// from and to are the same path (which handles all dirs cases) or the same inode (hardlinks to the same file)
		return true, nil
	}

	h1, err := state.PathHasher.Hash(from, false, true, hashTimestamp)
	if err != nil {
		return false, err
	}
	h2, err := state.PathHasher.Hash(to, false, true, hashTimestamp)
	return bytes.Equal(h1, h2), err
}

// Build builds a single filegroup file. Returns whether any files are changed or should be if there hadn't been an
// error.
func (builder *filegroupBuilder) Build(state *core.BuildState, target *core.BuildTarget, from, to string) (bool, error) {
	// Verify that the source actually exists. It is otherwise possible to get through here
	// without in certain circumstances (basically if another filegroup outputs the same file
	// from a genrule and has been built already, because we have it in builder.built).
	if !fs.PathExists(from) {
		return true, fmt.Errorf("Can't build %s: input %s does not exist", target, from)
	}
	builder.mutex.Lock()
	defer builder.mutex.Unlock()
	if changed, present := builder.built[to]; present {
		return changed, nil // File's already been built.
	}
	if same, err := isSameFileContent(state, target.HashLastModified(), from, to); err != nil {
		return false, err
	} else if same {
		// File exists already and is the same file. Nothing to do.
		builder.built[to] = false
		state.PathHasher.MoveHash(from, to, true)

		return false, nil
	}
	// Must actually build the file.
	if err := os.RemoveAll(to); err != nil {
		return true, err
	} else if err := fs.EnsureDir(to); err != nil {
		return true, err
	} else {
		// Copy source files when they're binary to avoid nuking their permissions.
		isSourceFile := !strings.HasPrefix(from, "plz-out/")
		if err := fs.RecursiveCopyOrLinkFile(from, to, target.OutMode(), !(target.IsBinary && isSourceFile), true); err != nil {
			return true, err
		}
	}
	builder.built[to] = true
	state.PathHasher.MoveHash(from, to, true)
	return true, nil
}

// buildFilegroup runs the manual build steps for a filegroup rule.
// We don't force this to be done in bash to avoid errors with maximum command lengths,
// and it's actually quite fiddly to get just so there.
func buildFilegroup(state *core.BuildState, target *core.BuildTarget) (bool, error) {
	if err := prepareDirectory(state.ProcessExecutor, target.OutDir(), false); err != nil {
		return true, err
	}
	changed := false
	outDir := target.OutDir()
	localSources := target.AllSourceLocalPaths(state.Graph)
	for i, source := range target.AllSourceFullPaths(state.Graph) {
		out := filepath.Join(outDir, localSources[i])
		fileChanged, err := theFilegroupBuilder.Build(state, target, source, out)
		if err != nil {
			return true, err
		}
		changed = changed || fileChanged
	}

	// Check if any of our srcs that are other built targets have changed.
	for _, bi := range target.AllSources() {
		if changed {
			break
		}
		l, ok := bi.Label()
		if ok && state.Graph.TargetOrDie(l).State() < core.Unchanged {
			changed = true
		}
	}

	if target.HasLabel("py") && !target.IsBinary {
		// Pre-emptively create __init__.py files so the outputs can be loaded dynamically.
		// It's a bit cheeky to do non-essential language-specific logic but this enables
		// a lot of relatively normal Python workflows.
		// Errors are deliberately ignored.
		if pkg := state.Graph.PackageByLabel(target.Label); pkg == nil || !pkg.HasOutput("__init__.py") {
			// Don't create this if someone else is going to create this in the package.
			createInitPy(outDir)
		}
	}
	if target.HasLabel("go") {
		// Create a dummy go.mod file so Go tooling ignores the contents of plz-out.
		goModOnce.Do(writeGoMod)
	}
	return changed, nil
}

// copyFilegroupHashes copies the hashes of the inputs of this filegroup to their outputs.
// This is a small optimisation to ensure we don't need to recalculate them unnecessarily.
func copyFilegroupHashes(state *core.BuildState, target *core.BuildTarget) {
	outDir := target.OutDir()
	localSources := target.AllSourceLocalPaths(state.Graph)
	for i, source := range target.AllSourceFullPaths(state.Graph) {
		if out := filepath.Join(outDir, localSources[i]); out != source {
			state.PathHasher.MoveHash(source, out, true)
		}
	}
}

func createInitPy(dir string) {
	initPy := filepath.Join(dir, "__init__.py")
	if core.PathExists(initPy) {
		return
	}
	if f, err := os.OpenFile(initPy, os.O_RDONLY|os.O_CREATE, 0444); err == nil {
		f.Close()
	}
	dir = filepath.Dir(dir)
	if dir != core.GenDir && dir != "." && !core.PathExists(filepath.Join(dir, "__init__.py")) {
		createInitPy(dir)
	}
}

func writeGoMod() {
	const contents = "module please-ignore  // Dummy module to exclude this directory from other tools\n"
	const filename = core.OutDir + "/go.mod"
	if !fs.PathExists(filename) {
		os.WriteFile(filename, []byte(contents), 0644)
	}
}
