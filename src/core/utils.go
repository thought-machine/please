package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"cli"
	"fs"
)

// RepoRoot is the root of the Please repository
var RepoRoot string

// initialWorkingDir is the directory we began in. Early on we chdir() to the repo root but for
// some things we need to remember this.
var initialWorkingDir string

// initialPackage is the initial subdir of the working directory, ie. what package did we start in.
// This is similar but not identical to initialWorkingDir.
var initialPackage string

// usingBazelWorkspace is true if we detected a Bazel WORKSPACE file to find our repo root.
var usingBazelWorkspace bool

// DirPermissions are the default permission bits we apply to directories.
const DirPermissions = os.ModeDir | 0775

// FindRepoRoot returns the root directory of the current repo and sets the initial working dir.
// It returns true if the repo root was found.
func FindRepoRoot() bool {
	initialWorkingDir, _ = os.Getwd()
	RepoRoot, initialPackage = getRepoRoot(ConfigFileName, false)
	return RepoRoot != ""
}

// MustFindRepoRoot returns the root directory of the current repo and sets the initial working dir.
// It dies on failure, although will fall back to looking for a Bazel WORKSPACE file first.
func MustFindRepoRoot() string {
	if RepoRoot != "" {
		return RepoRoot
	}
	if !FindRepoRoot() {
		RepoRoot, initialPackage = getRepoRoot("WORKSPACE", true)
		log.Warning("No .plzconfig file found to define the repo root.")
		log.Warning("Falling back to Bazel WORKSPACE at %s", path.Join(RepoRoot, "WORKSPACE"))
		usingBazelWorkspace = true
	}
	return RepoRoot
}

// InitialPackage returns a label corresponding to the initial package we started in.
func InitialPackage() []BuildLabel {
	// It's possible to start off in directories that aren't legal package names, because
	// our package naming is stricter than directory naming requirements.
	// In that case move up until we find somewhere we can run from.
	dir := initialPackage
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
func getRepoRoot(filename string, die bool) (string, string) {
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
	if die {
		log.Fatalf("Couldn't locate the repo root. Are you sure you're inside a plz repo?")
	}
	return "", ""
}

// StartedAtRepoRoot returns true if the build was initiated from the repo root.
// Used to provide slightly nicer output in some places.
func StartedAtRepoRoot() bool {
	return RepoRoot == initialWorkingDir
}

// RecursiveCopyFile copies either a single file or a directory.
// If 'link' is true then we'll hardlink files instead of copying them.
// If 'fallback' is true then we'll fall back to a copy if linking fails.
func RecursiveCopyFile(from string, to string, mode os.FileMode, link, fallback bool) error {
	if info, err := os.Stat(from); err == nil && info.IsDir() {
		return fs.WalkMode(from, func(name string, isDir bool, fileMode os.FileMode) error {
			dest := path.Join(to, name[len(from):])
			if isDir {
				return os.MkdirAll(dest, DirPermissions)
			}
			return fs.CopyOrLinkFile(name, dest, mode, link, fallback)
		})
	}
	return fs.CopyOrLinkFile(from, to, mode, link, fallback)
}

// safeBuffer is an io.Writer that ensures that only one thread writes to it at a time.
// This is important because we potentially have both stdout and stderr writing to the same
// buffer, and os.exec only guarantees goroutine-safety if both are the same writer, which in
// our case they're not (but are both ultimately causing writes to the same buffer)
type safeBuffer struct {
	sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(b []byte) (int, error) {
	sb.Lock()
	defer sb.Unlock()
	return sb.buf.Write(b)
}

func (sb *safeBuffer) Bytes() []byte {
	return sb.buf.Bytes()
}

func (sb *safeBuffer) String() string {
	return sb.buf.String()
}

// logProgress logs a message once a minute until the given context has expired.
// Used to provide some notion of progress while waiting for external commands.
func logProgress(ctx context.Context, target *BuildTarget) {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for i := 1; i < 1000000; i++ {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if i == 1 {
				log.Notice("%s still running after 1 minute %s", target.Label, progressMessage(target))
			} else {
				log.Notice("%s still running after %d minutes %s", target.Label, i, progressMessage(target))
			}
		}
	}
}

// progressMessage displays a progress message for a target, if it tracks progress.
func progressMessage(target *BuildTarget) string {
	if target.ShowProgress {
		return fmt.Sprintf("(%0.1f%% done)", target.Progress)
	}
	return ""
}

// ExecWithTimeout runs an external command with a timeout.
// If the command times out the returned error will be a context.DeadlineExceeded error.
// If showOutput is true then output will be printed to stderr as well as returned.
// It returns the stdout only, combined stdout and stderr and any error that occurred.
func ExecWithTimeout(target *BuildTarget, dir string, env []string, timeout time.Duration, defaultTimeout cli.Duration, showOutput, attachStdStreams bool, argv []string) ([]byte, []byte, error) {
	if timeout == 0 {
		if defaultTimeout == 0 {
			timeout = 10 * time.Minute
		} else {
			timeout = time.Duration(defaultTimeout)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := ExecCommand(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = env

	var out bytes.Buffer
	var outerr safeBuffer
	if showOutput {
		cmd.Stdout = io.MultiWriter(os.Stderr, &out, &outerr)
		cmd.Stderr = io.MultiWriter(os.Stderr, &outerr)
	} else {
		cmd.Stdout = io.MultiWriter(&out, &outerr)
		cmd.Stderr = &outerr
	}
	if target != nil && target.ShowProgress {
		cmd.Stdout = newProgressWriter(target, cmd.Stdout)
		cmd.Stderr = newProgressWriter(target, cmd.Stderr)
	}
	if attachStdStreams {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if target != nil {
		go logProgress(ctx, target)
	}
	// Start the command, wait for the timeout & then kill it.
	// We deliberately don't use CommandContext because it will only send SIGKILL which
	// child processes can't handle themselves.
	err := cmd.Start()
	if err != nil {
		return nil, nil, err
	}
	ch := make(chan error)
	go runCommand(cmd, ch)
	select {
	case err = <-ch:
		// Do nothing.
	case <-time.After(timeout):
		// Send a relatively gentle signal that it can catch.
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Notice("Failed to kill subprocess: %s", err)
		}
		time.Sleep(10 * time.Millisecond)
		// Send a more forceful signal.
		cmd.Process.Kill()
		err = fmt.Errorf("Timeout exceeded: %s", outerr.String())
	}
	return out.Bytes(), outerr.Bytes(), err
}

// runCommand runs a command and signals on the given channel when it's done.
func runCommand(cmd *exec.Cmd, ch chan error) {
	ch <- cmd.Wait()
}

// ExecWithTimeoutShell runs an external command within a Bash shell.
// Other arguments are as ExecWithTimeout.
// Note that the command is deliberately a single string.
func ExecWithTimeoutShell(state *BuildState, target *BuildTarget, dir string, env []string, timeout time.Duration, defaultTimeout cli.Duration, showOutput bool, cmd string, sandbox bool) ([]byte, []byte, error) {
	return ExecWithTimeoutShellStdStreams(state, target, dir, env, timeout, defaultTimeout, showOutput, cmd, sandbox, false)
}

// ExecWithTimeoutShellStdStreams is as ExecWithTimeoutShell but optionally attaches stdin to the subprocess.
func ExecWithTimeoutShellStdStreams(state *BuildState, target *BuildTarget, dir string, env []string, timeout time.Duration, defaultTimeout cli.Duration, showOutput bool, cmd string, sandbox, attachStdStreams bool) ([]byte, []byte, error) {
	c := append([]string{"bash", "-u", "-o", "pipefail", "-c"}, cmd)
	// Runtime check is a little ugly, but we know this only works on Linux right now.
	if sandbox && runtime.GOOS == "linux" {
		tool, err := LookPath(state.Config.Build.PleaseSandboxTool, state.Config.Build.Path)
		if err != nil {
			return nil, nil, err
		}
		c = append([]string{tool}, c...)
	}
	return ExecWithTimeout(target, dir, env, timeout, defaultTimeout, showOutput, attachStdStreams, c)
}

// ExecWithTimeoutSimple runs an external command with a timeout.
// It's a simpler version of ExecWithTimeout that gives less control.
func ExecWithTimeoutSimple(timeout cli.Duration, cmd ...string) ([]byte, error) {
	_, out, err := ExecWithTimeout(nil, "", nil, time.Duration(timeout), timeout, false, false, cmd)
	return out, err
}

// A SourcePair represents a source file with its source and temporary locations.
// This isn't typically used much by callers; it's just useful to have a single type for channels.
type SourcePair struct{ Src, Tmp string }

// IterSources returns all the sources for a function, allowing for sources that are other rules
// and rules that require transitive dependencies.
// Yielded values are pairs of the original source location and its temporary location for this rule.
func IterSources(graph *BuildGraph, target *BuildTarget) <-chan SourcePair {
	ch := make(chan SourcePair)
	done := map[BuildLabel]bool{}
	donePaths := map[string]bool{}
	tmpDir := target.TmpDir()
	var inner func(dependency *BuildTarget)
	inner = func(dependency *BuildTarget) {
		sources := dependency.AllSources()
		if target == dependency {
			// This is the current build rule, so link its sources.
			for _, source := range sources {
				for _, providedSource := range recursivelyProvideSource(graph, target, source) {
					fullPaths := providedSource.FullPaths(graph)
					for i, sourcePath := range providedSource.Paths(graph) {
						tmpPath := path.Join(tmpDir, sourcePath)
						ch <- SourcePair{fullPaths[i], tmpPath}
						donePaths[tmpPath] = true
					}
				}
			}
		} else {
			// This is a dependency of the rule, so link its outputs.
			outDir := dependency.OutDir()
			for _, dep := range dependency.Outputs() {
				depPath := path.Join(outDir, dep)
				pkgName := dependency.Subrepo.MakeRelativeName(dependency.Label.PackageName)
				tmpPath := path.Join(tmpDir, pkgName, dep)
				if !donePaths[tmpPath] {
					ch <- SourcePair{depPath, tmpPath}
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
		// All the sources of this rule now count as done.
		for _, source := range sources {
			if label := source.Label(); label != nil && dependency.IsSourceOnlyDep(*label) {
				done[*label] = true
			}
		}

		done[dependency.Label] = true
		if target == dependency || (target.NeedsTransitiveDependencies && !dependency.OutputIsComplete) {
			for _, dep := range dependency.BuildDependencies() {
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
		inner(target)
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
func recursivelyProvideSource(graph *BuildGraph, target *BuildTarget, src BuildInput) []BuildInput {
	if label := src.nonOutputLabel(); label != nil {
		dep := graph.TargetOrDie(*label)
		provided := recursivelyProvideFor(graph, target, target, dep.Label)
		ret := make([]BuildInput, len(provided))
		for i, p := range provided {
			ret[i] = p
		}
		return ret
	}
	return []BuildInput{src}
}

// IterRuntimeFiles yields all the runtime files for a rule (outputs & data files), similar to above.
func IterRuntimeFiles(graph *BuildGraph, target *BuildTarget, absoluteOuts bool) <-chan SourcePair {
	done := map[string]bool{}
	ch := make(chan SourcePair)

	makeOut := func(out string) string {
		if absoluteOuts {
			return path.Join(RepoRoot, target.TestDir(), out)
		}
		return out
	}

	pushOut := func(src, out string) {
		out = makeOut(out)
		if !done[out] {
			ch <- SourcePair{src, out}
			done[out] = true
		}
	}

	var inner func(*BuildTarget)
	inner = func(target *BuildTarget) {
		outDir := target.OutDir()
		for _, out := range target.Outputs() {
			pushOut(path.Join(outDir, out), out)
		}
		for _, data := range target.Data {
			var subrepo *Subrepo
			label := data.Label()
			if label != nil {
				subrepo = graph.TargetOrDie(*label).Subrepo
			}
			fullPaths := data.FullPaths(graph)
			for i, dataPath := range data.Paths(graph) {
				pushOut(fullPaths[i], subrepo.MakeRelativeName(dataPath))
			}
			if label != nil {
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
				if label := source.nonOutputLabel(); label == nil {
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
	return RecursiveCopyFile(sourcePath, tmpPath, 0, true, true)
}

// PrepareSourcePair prepares a source file for a build.
func PrepareSourcePair(pair SourcePair) error {
	if path.IsAbs(pair.Src) {
		return PrepareSource(pair.Src, pair.Tmp)
	}
	return PrepareSource(path.Join(RepoRoot, pair.Src), pair.Tmp)
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
	return "", fmt.Errorf("%s not found in PATH %s", filename, strings.Join(paths, ":"))
}

// AsyncDeleteDir deletes a directory asynchronously.
// First it renames the directory to something temporary and then forks to delete it.
// The rename is done synchronously but the actual deletion is async (after fork) so
// you don't have to wait for large directories to be removed.
// Conversely there is obviously no guarantee about at what point it will actually cease to
// be on disk any more.
func AsyncDeleteDir(dir string) error {
	rm, err := exec.LookPath("rm")
	if err != nil {
		return err
	} else if !PathExists(dir) {
		return nil // not an error, just don't need to do anything.
	}
	newDir, err := moveDir(dir)
	if err != nil {
		return err
	}
	// Note that we can't fork() directly and continue running Go code, but ForkExec() works okay.
	// Hence why we're using rm rather than fork() + os.RemoveAll.
	_, err = syscall.ForkExec(rm, []string{rm, "-rf", newDir}, nil)
	return err
}

// moveDir moves a directory to a new location and returns that new location.
func moveDir(dir string) (string, error) {
	b := make([]byte, 16)
	rand.Read(b)
	name := path.Join(path.Dir(dir), ".plz_clean_"+hex.EncodeToString(b))
	log.Notice("Moving %s to %s", dir, name)
	return name, os.Rename(dir, name)
}

// PathExists is an alias to fs.PathExists.
// TODO(peterebden): Remove and migrate everything over.
func PathExists(filename string) bool {
	return fs.PathExists(filename)
}
