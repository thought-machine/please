// Package build houses the core functionality for actually building targets.
package build

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"gopkg.in/op/go-logging.v1"

	"core"
	"fs"
	"metrics"
)

var log = logging.MustGetLogger("build")

// Type that indicates that we're stopping the build of a target in a nonfatal way.
var errStop = fmt.Errorf("stopping build")

// goDirOnce guards the creation of plz-out/go, which we only attempt once per process.
var goDirOnce sync.Once

// httpClient is the shared http client that we use for fetching remote files.
var httpClient http.Client

// Build implements the core logic for building a single target.
func Build(tid int, state *core.BuildState, label core.BuildLabel) {
	start := time.Now()
	target := state.Graph.TargetOrDie(label)
	state = state.ForTarget(target)
	target.SetState(core.Building)
	if err := buildTarget(tid, state, target); err != nil {
		if err == errStop {
			target.SetState(core.Stopped)
			state.LogBuildResult(tid, target.Label, core.TargetBuildStopped, "Build stopped")
			return
		}
		state.LogBuildError(tid, label, core.TargetBuildFailed, err, "Build failed: %s", err)
		if err := RemoveOutputs(target); err != nil {
			log.Errorf("Failed to remove outputs for %s: %s", target.Label, err)
		}
		target.SetState(core.Failed)
		return
	}
	metrics.Record(target, time.Since(start))

	// Add any of the reverse deps that are now fully built to the queue.
	for _, reverseDep := range state.Graph.ReverseDependencies(target) {
		if reverseDep.State() == core.Active && state.Graph.AllDepsBuilt(reverseDep) && reverseDep.SyncUpdateState(core.Active, core.Pending) {
			state.AddPendingBuild(reverseDep.Label, false)
		}
	}
	if target.IsTest && state.NeedTests {
		state.AddPendingTest(target.Label)
	}
	state.Parser.UndeferAnyParses(state, target)
}

// Builds a single target
func buildTarget(tid int, state *core.BuildState, target *core.BuildTarget) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%s", r)
			}
		}
	}()

	if err := target.CheckDependencyVisibility(state); err != nil {
		return err
	}
	// We can't do this check until build time, until then we don't know what all the outputs
	// will be (eg. for filegroups that collect outputs of other rules).
	if err := target.CheckDuplicateOutputs(); err != nil {
		return err
	}
	// This must run before we can leave this function successfully by any path.
	if target.PreBuildFunction != nil {
		log.Debug("Running pre-build function for %s", target.Label)
		if err := state.Parser.RunPreBuildFunction(tid, state, target); err != nil {
			return err
		}
		log.Debug("Finished pre-build function for %s", target.Label)
	}
	state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Preparing...")
	var postBuildOutput string
	if state.PrepareOnly && state.IsOriginalTarget(target.Label) {
		if target.IsFilegroup {
			return fmt.Errorf("Filegroup targets don't have temporary directories")
		}
		if err := prepareDirectories(target); err != nil {
			return err
		}
		if err := prepareSources(state.Graph, target); err != nil {
			return err
		}
		return errStop
	}
	if target.IsHashFilegroup {
		updateHashFilegroupPaths(state, target)
	}
	if !needsBuilding(state, target, false) {
		log.Debug("Not rebuilding %s, nothing's changed", target.Label)
		if postBuildOutput, err = runPostBuildFunctionIfNeeded(tid, state, target, ""); err != nil {
			log.Warning("Missing post-build output for %s; will rebuild.", target.Label)
		} else {
			// If a post-build function ran it may modify the rule definition. In that case we
			// need to check again whether the rule needs building.
			if target.PostBuildFunction == nil || !needsBuilding(state, target, true) {
				if target.IsFilegroup {
					// Small optimisation to ensure we don't need to rehash things unnecessarily.
					copyFilegroupHashes(state, target)
				}
				target.SetState(core.Reused)
				state.LogBuildResult(tid, target.Label, core.TargetCached, "Unchanged")
				return nil // Nothing needs to be done.
			}
			log.Debug("Rebuilding %s after post-build function", target.Label)
		}
	}
	oldOutputHash, outputHashErr := OutputHash(target)
	if target.IsFilegroup {
		log.Debug("Building %s...", target.Label)
		if err := buildFilegroup(tid, state, target); err != nil {
			return err
		} else if newOutputHash, err := calculateAndCheckRuleHash(state, target); err != nil {
			return err
		} else if !bytes.Equal(newOutputHash, oldOutputHash) {
			target.SetState(core.Built)
			state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
		} else {
			target.SetState(core.Unchanged)
			state.LogBuildResult(tid, target.Label, core.TargetCached, "Unchanged")
		}
		return nil
	}
	if err := prepareDirectories(target); err != nil {
		return fmt.Errorf("Error preparing directories for %s: %s", target.Label, err)
	}

	// Similarly to the createInitPy special-casing, this is not very nice, but makes it
	// rather easier to have a consistent GOPATH setup.
	if target.HasLabel("go") {
		goDirOnce.Do(createPlzOutGo)
	}

	retrieveArtifacts := func() bool {
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Checking cache...")
		if _, retrieved := retrieveFromCache(state, target); retrieved {
			log.Debug("Retrieved artifacts for %s from cache", target.Label)
			checkLicences(state, target)
			newOutputHash, err := calculateAndCheckRuleHash(state, target)
			if err != nil { // Most likely hash verification failure
				log.Warning("Error retrieving cached artifacts for %s: %s", target.Label, err)
				RemoveOutputs(target)
				return false
			} else if outputHashErr != nil || !bytes.Equal(oldOutputHash, newOutputHash) {
				target.SetState(core.Cached)
				state.LogBuildResult(tid, target.Label, core.TargetCached, "Cached")
			} else {
				target.SetState(core.Unchanged)
				state.LogBuildResult(tid, target.Label, core.TargetCached, "Cached (unchanged)")
			}
			return true // got from cache
		}
		return false
	}
	cacheKey := mustShortTargetHash(state, target)
	if state.Cache != nil {
		// Note that ordering here is quite sensitive since the post-build function can modify
		// what we would retrieve from the cache.
		if target.PostBuildFunction != nil {
			log.Debug("Checking for post-build output file for %s in cache...", target.Label)
			if state.Cache.RetrieveExtra(target, cacheKey, target.PostBuildOutputFileName()) {
				if postBuildOutput, err = runPostBuildFunctionIfNeeded(tid, state, target, postBuildOutput); err != nil {
					panic(err)
				}
				if retrieveArtifacts() {
					return nil
				}
			}
		} else if retrieveArtifacts() {
			return nil
		}
	}
	if err := target.CheckSecrets(); err != nil {
		return err
	}
	if err := prepareSources(state.Graph, target); err != nil {
		return fmt.Errorf("Error preparing sources for %s: %s", target.Label, err)
	}

	state.LogBuildResult(tid, target.Label, core.TargetBuilding, target.BuildingDescription)
	out, err := buildMaybeRemotely(state, target, cacheKey)
	if err != nil {
		return err
	}
	if target.PostBuildFunction != nil {
		out = bytes.TrimSpace(out)
		if err := runPostBuildFunction(tid, state, target, string(out), postBuildOutput); err != nil {
			return err
		}
		storePostBuildOutput(state, target, out)
	}
	checkLicences(state, target)
	state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Collecting outputs...")
	extraOuts, outputsChanged, err := moveOutputs(state, target)
	if err != nil {
		return fmt.Errorf("Error moving outputs for target %s: %s", target.Label, err)
	}
	if _, err = calculateAndCheckRuleHash(state, target); err != nil {
		return err
	}
	if outputsChanged {
		target.SetState(core.Built)
	} else {
		target.SetState(core.Unchanged)
	}
	if state.Cache != nil {
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Storing...")
		newCacheKey := mustShortTargetHash(state, target)
		if target.PostBuildFunction != nil {
			if !bytes.Equal(newCacheKey, cacheKey) {
				// NB. Important this is stored with the earlier hash - if we calculate the hash
				//     now, it might be different, and we could of course never retrieve it again.
				state.Cache.StoreExtra(target, cacheKey, target.PostBuildOutputFileName())
			} else {
				extraOuts = append(extraOuts, target.PostBuildOutputFileName())
			}
		}
		state.Cache.Store(target, newCacheKey, extraOuts...)
	}
	// Clean up the temporary directory once it's done.
	if state.CleanWorkdirs {
		if err := os.RemoveAll(target.TmpDir()); err != nil {
			log.Warning("Failed to remove temporary directory for %s: %s", target.Label, err)
		}
	}
	if outputsChanged {
		state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
	} else {
		state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built (unchanged)")
	}
	return nil
}

// runBuildCommand runs the actual command to build a target.
// On success it returns the stdout of the target, otherwise an error.
func runBuildCommand(state *core.BuildState, target *core.BuildTarget, command string, inputHash []byte) ([]byte, error) {
	if target.IsRemoteFile {
		return nil, fetchRemoteFile(state, target)
	}
	env := core.StampedBuildEnvironment(state, target, false, inputHash)
	log.Debug("Building target %s\nENVIRONMENT:\n%s\n%s", target.Label, env, command)
	out, combined, err := core.ExecWithTimeoutShell(state, target, target.TmpDir(), env, target.BuildTimeout, state.Config.Build.Timeout, state.ShowAllOutput, command, target.Sandbox)
	if err != nil {
		if state.Verbosity >= 4 {
			return nil, fmt.Errorf("Error building target %s: %s\nENVIRONMENT:\n%s\n%s\n%s",
				target.Label, err, env, target.GetCommand(state), combined)
		}
		return nil, fmt.Errorf("Error building target %s: %s\n%s", target.Label, err, combined)
	}
	return out, nil
}

// Prepares the output directories for a target
func prepareDirectories(target *core.BuildTarget) error {
	if err := prepareDirectory(target.TmpDir(), true); err != nil {
		return err
	}
	if err := prepareDirectory(target.OutDir(), false); err != nil {
		return err
	}
	// Nicety for the build rules: create any directories that it's
	// declared it'll create files in.
	for _, out := range target.Outputs() {
		if dir := path.Dir(out); dir != "." {
			outPath := path.Join(target.TmpDir(), dir)
			if !core.PathExists(outPath) {
				if err := os.MkdirAll(outPath, core.DirPermissions); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func prepareDirectory(directory string, remove bool) error {
	if remove && core.PathExists(directory) {
		if err := os.RemoveAll(directory); err != nil {
			return err
		}
	}
	err := os.MkdirAll(directory, core.DirPermissions)
	if err != nil && checkForStaleOutput(directory, err) {
		err = os.MkdirAll(directory, core.DirPermissions)
	}
	return err
}

// Symlinks the source files of this rule into its temp directory.
func prepareSources(graph *core.BuildGraph, target *core.BuildTarget) error {
	for source := range core.IterSources(graph, target) {
		if err := core.PrepareSourcePair(source); err != nil {
			return err
		}
	}
	return nil
}

func moveOutputs(state *core.BuildState, target *core.BuildTarget) ([]string, bool, error) {
	// Before we write any outputs, we must remove the old hash file to avoid it being
	// left in an inconsistent state.
	if err := os.RemoveAll(ruleHashFileName(target)); err != nil {
		return nil, true, err
	}
	changed := false
	tmpDir := target.TmpDir()
	outDir := target.OutDir()
	for _, output := range target.Outputs() {
		tmpOutput := path.Join(tmpDir, output)
		realOutput := path.Join(outDir, output)
		if !core.PathExists(tmpOutput) {
			return nil, true, fmt.Errorf("Rule %s failed to create output %s", target.Label, tmpOutput)
		}
		outputChanged, err := moveOutput(target, tmpOutput, realOutput)
		if err != nil {
			return nil, true, err
		}
		changed = changed || outputChanged
	}
	if changed {
		log.Debug("Outputs for %s have changed", target.Label)
	} else {
		log.Debug("Outputs for %s are unchanged", target.Label)
	}
	// Optional outputs get moved but don't contribute to the hash or for incrementality.
	// Glob patterns are supported on these.
	extraOuts := []string{}
	for _, output := range fs.Glob(state.Config.Parse.BuildFileName, tmpDir, target.OptionalOutputs, nil, nil, true) {
		log.Debug("Discovered optional output %s", output)
		tmpOutput := path.Join(tmpDir, output)
		realOutput := path.Join(outDir, output)
		if _, err := moveOutput(target, tmpOutput, realOutput); err != nil {
			return nil, changed, err
		}
		extraOuts = append(extraOuts, output)
	}
	return extraOuts, changed, nil
}

func moveOutput(target *core.BuildTarget, tmpOutput, realOutput string) (bool, error) {
	// hash the file
	newHash, err := pathHash(tmpOutput, false)
	if err != nil {
		return true, err
	}
	if fs.PathExists(realOutput) {
		if oldHash, err := pathHash(realOutput, false); err != nil {
			return true, err
		} else if bytes.Equal(oldHash, newHash) {
			// We already have the same file in the current location. Don't bother moving it.
			log.Debug("Checking %s vs. %s, hashes match", tmpOutput, realOutput)
			return false, nil
		}
		if err := os.RemoveAll(realOutput); err != nil {
			return true, err
		}
	}
	movePathHash(tmpOutput, realOutput, false)
	// Check if we need a directory for this output.
	dir := path.Dir(realOutput)
	if !core.PathExists(dir) {
		if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
			return true, err
		}
	}
	// If the output file is in plz-out/tmp we can just move it to save time, otherwise we need
	// to copy so we don't move files from other directories.
	if strings.HasPrefix(tmpOutput, target.TmpDir()) {
		if err := os.Rename(tmpOutput, realOutput); err != nil {
			return true, err
		}
	} else {
		if err := core.RecursiveCopyFile(tmpOutput, realOutput, target.OutMode(), false, false); err != nil {
			return true, err
		}
	}
	if target.IsBinary {
		if err := os.Chmod(realOutput, target.OutMode()); err != nil {
			return true, err
		}
	}
	return true, nil
}

// RemoveOutputs removes all generated outputs for a rule.
func RemoveOutputs(target *core.BuildTarget) error {
	if err := os.Remove(ruleHashFileName(target)); err != nil && !os.IsNotExist(err) {
		if checkForStaleOutput(ruleHashFileName(target), err) {
			return RemoveOutputs(target) // try again
		}
		return err
	}
	for _, output := range target.Outputs() {
		if err := os.RemoveAll(path.Join(target.OutDir(), output)); err != nil {
			return err
		}
	}
	return nil
}

// checkForStaleOutput removes any parents of a file that are files themselves.
// This is a fix for a specific case where there are old file outputs in plz-out which
// have the same name as part of a package path.
// It returns true if something was removed.
func checkForStaleOutput(filename string, err error) bool {
	if perr, ok := err.(*os.PathError); ok && perr.Err.Error() == "not a directory" {
		for dir := path.Dir(filename); dir != "." && dir != "/" && path.Base(dir) != "plz-out"; dir = path.Dir(filename) {
			if fs.FileExists(dir) {
				log.Warning("Removing %s which appears to be a stale output file", dir)
				os.Remove(dir)
				return true
			}
		}
	}
	return false
}

// calculateAndCheckRuleHash checks the output hash for a rule.
func calculateAndCheckRuleHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	hash, err := OutputHash(target)
	if err != nil {
		return nil, err
	}
	if err = checkRuleHashes(target, hash); err != nil {
		if state.NeedHashesOnly && (state.IsOriginalTarget(target.Label) || state.IsOriginalTarget(target.Label.Parent())) {
			return nil, errStop
		} else if state.VerifyHashes {
			return nil, err
		} else {
			log.Warning("%s", err)
		}
	}
	if err := writeRuleHashFile(state, target); err != nil {
		return nil, fmt.Errorf("Attempting to create hash file: %s", err)
	}
	return hash, nil
}

// OutputHash calculates the hash of a target's outputs.
func OutputHash(target *core.BuildTarget) ([]byte, error) {
	h := sha1.New()
	for _, output := range target.Outputs() {
		// NB. Always force a recalculation of the output hashes here. Memoisation is not
		//     useful because by definition we are rebuilding a target, and can actively hurt
		//     in cases where we compare the retrieved cache artifacts with what was there before.
		filename := path.Join(target.OutDir(), output)
		h2, err := pathHash(filename, true)
		if err != nil {
			return nil, err
		}
		h.Write(h2)
		// Record the name of the file too, but not if the rule has hash verification
		// (because this will change the hashes, and the cases it fixes are relatively rare
		// and generally involve things like hash_filegroup that doesn't have hashes set).
		// TODO(pebers): Find some more elegant way of unifying this behaviour.
		if len(target.Hashes) == 0 {
			h.Write([]byte(filename))
		}
	}
	return h.Sum(nil), nil
}

// mustOutputHash calculates the hash of a target's outputs. It panics on any errors.
func mustOutputHash(target *core.BuildTarget) []byte {
	hash, err := OutputHash(target)
	if err != nil {
		panic(err)
	}
	return hash
}

// Verify the hash of output files for a rule match the ones set on it.
func checkRuleHashes(target *core.BuildTarget, hash []byte) error {
	if len(target.Hashes) == 0 {
		return nil // nothing to check
	}
	hashStr := hex.EncodeToString(hash)
	for _, okHash := range target.Hashes {
		// Hashes can have an arbitrary label prefix. Strip it off if present.
		if index := strings.LastIndexByte(okHash, ':'); index != -1 {
			okHash = strings.TrimSpace(okHash[index+1:])
		}
		if okHash == hashStr {
			return nil
		}
	}
	if len(target.Hashes) == 1 {
		return fmt.Errorf("Bad output hash for rule %s: was %s but expected %s",
			target.Label, hashStr, target.Hashes[0])
	}
	return fmt.Errorf("Bad output hash for rule %s: was %s but expected one of [%s]",
		target.Label, hashStr, strings.Join(target.Hashes, ", "))
}

func retrieveFromCache(state *core.BuildState, target *core.BuildTarget) ([]byte, bool) {
	hash := mustShortTargetHash(state, target)
	return hash, state.Cache.Retrieve(target, hash)
}

// Runs the post-build function for a target if it's got one.
func runPostBuildFunctionIfNeeded(tid int, state *core.BuildState, target *core.BuildTarget, prevOutput string) (string, error) {
	if target.PostBuildFunction != nil {
		out, err := loadPostBuildOutput(state, target)
		if err != nil {
			return "", err
		}
		return out, runPostBuildFunction(tid, state, target, out, prevOutput)
	}
	return "", nil
}

// Runs the post-build function for a target.
// In some cases it may have already run; if so we compare the previous output and warn
// if the two differ (they must be deterministic to ensure it's a pure function, since there
// are a few different paths through here and we guarantee to only run them once).
func runPostBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget, output, prevOutput string) error {
	if prevOutput != "" {
		if output != prevOutput {
			log.Warning("The build output for %s differs from what we got back from the cache earlier.\n"+
				"This implies your target's output is nondeterministic; Please won't re-run the\n"+
				"post-build function, which will *probably* be okay, but Please can't be sure.\n"+
				"See https://github.com/thought-machine/please/issues/113 for more information.", target.Label)
			log.Debug("Cached build output for %s: %s\n\nNew build output: %s", target.Label, prevOutput, output)
		}
		return nil
	}
	return state.Parser.RunPostBuildFunction(tid, state, target, output)
}

// checkLicences checks the licences for the target match what we've accepted / rejected in the config
// and panics if they don't match.
func checkLicences(state *core.BuildState, target *core.BuildTarget) {
	for _, licence := range target.Licences {
		for _, reject := range state.Config.Licences.Reject {
			if strings.EqualFold(reject, licence) {
				panic(fmt.Sprintf("Target %s is licensed %s, which is explicitly rejected for this repository", target.Label, licence))
			}
		}
		for _, accept := range state.Config.Licences.Accept {
			if strings.EqualFold(accept, licence) {
				log.Info("Licence %s is accepted in this repository", licence)
				return // Note licences are assumed to be an 'or', ie. any one of them can be accepted.
			}
		}
	}
	if len(target.Licences) > 0 && len(state.Config.Licences.Accept) > 0 {
		panic(fmt.Sprintf("None of the licences for %s are accepted in this repository: %s", target.Label, strings.Join(target.Licences, ", ")))
	}
}

// createPlzOutGo creates a directory plz-out/go that contains src / pkg links which
// make it easier to set up one's GOPATH appropriately.
func createPlzOutGo() {
	dir := path.Join(core.RepoRoot, core.OutDir, "go")
	genDir := path.Join(core.RepoRoot, core.GenDir)
	srcDir := path.Join(dir, "src")
	pkgDir := path.Join(dir, "pkg")
	archDir := path.Join(pkgDir, runtime.GOOS+"_"+runtime.GOARCH)
	if err := os.MkdirAll(pkgDir, core.DirPermissions); err != nil {
		log.Warning("Failed to create %s: %s", pkgDir, err)
		return
	}
	symlinkIfNotExists(genDir, srcDir)
	symlinkIfNotExists(genDir, archDir)
}

// symlinkIfNotExists creates newDir as a link to oldDir if it doesn't already exist.
func symlinkIfNotExists(oldDir, newDir string) {
	if !core.PathExists(newDir) {
		if err := os.Symlink(oldDir, newDir); err != nil && !os.IsExist(err) {
			log.Warning("Failed to create %s: %s", newDir, err)
		}
	}
}

// fetchRemoteFile fetches a remote file from a URL.
// This is a builtin for better efficiency and more control over the whole process.
func fetchRemoteFile(state *core.BuildState, target *core.BuildTarget) error {
	if err := prepareDirectory(target.OutDir(), false); err != nil {
		return err
	} else if err := prepareDirectory(target.TmpDir(), false); err != nil {
		return err
	} else if err := os.RemoveAll(ruleHashFileName(target)); err != nil {
		return err
	}
	httpClient.Timeout = time.Duration(state.Config.Build.Timeout) // Can't set this when we init the client because config isn't loaded then.
	var err error
	for _, src := range target.Sources {
		if e := fetchOneRemoteFile(state, target, string(src.(core.URLLabel))); e != nil {
			err = multierror.Append(err, e)
		} else {
			return nil
		}
	}
	return err
}

func fetchOneRemoteFile(state *core.BuildState, target *core.BuildTarget, url string) error {
	env := core.BuildEnvironment(state, target, false)
	url = os.Expand(url, env.ReplaceEnvironment)
	tmpPath := path.Join(target.TmpDir(), target.Outputs()[0])
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Error retrieving %s: %s", url, resp.Status)
	}
	var r io.Reader = resp.Body
	if length := resp.Header.Get("Content-Length"); length != "" {
		if i, err := strconv.Atoi(length); err == nil {
			r = &progressReader{Reader: resp.Body, Target: target, Total: float32(i)}
		}
	}
	target.ShowProgress = true // Required for it to actually display
	h := sha1.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		return err
	}
	setPathHash(tmpPath, h.Sum(nil))
	return f.Close()
}

// A progressReader tracks progress from a HTTP response and marks it on the given target.
type progressReader struct {
	Reader      io.Reader
	Target      *core.BuildTarget
	Done, Total float32
}

// Read implements the io.Reader interface
func (r *progressReader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	r.Done += float32(n)
	r.Target.Progress = 100.0 * r.Done / r.Total
	return n, err
}
