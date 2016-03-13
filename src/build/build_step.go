// The build package houses the core functionality for actually building targets.
package build

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"core"
	"parse"
)

var log = logging.MustGetLogger("build")

// Type that indicates that we're stopping the build of a target in a nonfatal way.
type stopTarget string

func (err stopTarget) Error() string { return string(err) }

func Build(tid int, state *core.BuildState, label core.BuildLabel) {
	target := state.Graph.TargetOrDie(label)
	target.SetState(core.Building)
	if err := buildTarget(tid, state, target); err != nil {
		if _, ok := err.(stopTarget); ok {
			target.SetState(core.Stopped)
			state.LogBuildResult(tid, target.Label, core.TargetBuildStopped, "Stopped")
			return
		}
		state.LogBuildError(tid, label, core.TargetBuildFailed, err, "Build failed: %s", err)
		if err := removeOutputs(target); err != nil {
			log.Errorf("Failed to remove outputs for %s: %s", target.Label, err)
		}
		target.SetState(core.Failed)
		return
	}

	// Add any of the reverse deps that are now fully built to the queue.
	for _, reverseDep := range state.Graph.ReverseDependencies(target) {
		if reverseDep.State() == core.Active && state.Graph.AllDepsBuilt(reverseDep) && reverseDep.SyncUpdateState(core.Active, core.Pending) {
			state.AddPendingBuild(reverseDep.Label)
		}
	}
	if target.IsTest && state.NeedTests {
		state.AddPendingTest(target.Label)
	}
	parse.UndeferAnyParses(state, target)
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

	if err := target.CheckDependencyVisibility(); err != nil {
		return err
	}
	// We can't do this check until build time, until then we don't know what all the outputs
	// will be (eg. for filegroups that collect outputs of other rules).
	if err := target.CheckDuplicateOutputs(); err != nil {
		return err
	}
	state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Preparing...")
	// This must run before we can leave this function successfully by any path.
	if target.PreBuildFunction != 0 {
		if err := parse.RunPreBuildFunction(tid, state, target); err != nil {
			return err
		}
	}
	if !needsBuilding(state, target, false) {
		log.Debug("Not rebuilding %s, nothing's changed", target.Label)
		runPostBuildFunctionIfNeeded(tid, state, target)
		// If a post-build function ran it may modify the rule definition. In that case we
		// need to check again whether the rule needs building.
		if target.PostBuildFunction == 0 || !needsBuilding(state, target, true) {
			target.SetState(core.Unchanged)
			state.LogBuildResult(tid, target.Label, core.TargetCached, "Unchanged")
			return nil // Nothing needs to be done.
		} else {
			log.Debug("Rebuilding %s after post-build function", target.Label)
		}
	}
	if err := prepareDirectories(target); err != nil {
		return fmt.Errorf("Error preparing directories for %s: %s", target.Label, err)
	}

	retrieveArtifacts := func() bool {
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Checking cache...")
		if _, retrieved := retrieveFromCache(state, target); retrieved {
			log.Debug("Retrieved artifacts for %s from cache", target.Label)
			if calculateAndCheckRuleHash(state, target) {
				target.SetState(core.Built)
			} else {
				target.SetState(core.Unchanged)
			}
			checkLicences(state, target)
			state.LogBuildResult(tid, target.Label, core.TargetCached, "Cached")
			return true // got from cache
		}
		return false
	}

	cacheKey := mustShortTargetHash(state, target)
	if state.Cache != nil && !target.SkipCache {
		// Note that ordering here is quite sensitive since the post-build function can modify
		// what we would retrieve from the cache.
		if target.PostBuildFunction != 0 {
			log.Debug("Checking for post-build output file for %s in cache...", target.Label)
			if (*state.Cache).RetrieveExtra(target, cacheKey, core.PostBuildOutputFileName(target)) {
				runPostBuildFunctionIfNeeded(tid, state, target)
				if retrieveArtifacts() {
					return nil
				}
			}
		} else if retrieveArtifacts() {
			return nil
		}
	}
	if err := prepareSources(state.Graph, target, target, map[core.BuildLabel]bool{}); err != nil {
		return fmt.Errorf("Error preparing sources for %s: %s", target.Label, err)
	}
	state.LogBuildResult(tid, target.Label, core.TargetBuilding, target.BuildingDescription)
	replacedCmd := replaceSequences(target)
	cmd := exec.Command("bash", "-c", replacedCmd)
	cmd.Dir = target.TmpDir()
	cmd.Env = core.BuildEnvironment(state, target, false)
	log.Debug("Building target %s\nENVIRONMENT:\n%s\n%s", target.Label, strings.Join(cmd.Env, "\n"), replacedCmd)
	if state.PrintCommands {
		log.Notice("Building %s: %s", target.Label, replacedCmd)
	}
	out, err := core.ExecWithTimeout(cmd, target.BuildTimeout, state.Config.Build.Timeout)
	if err != nil {
		if state.Verbosity >= 4 {
			return fmt.Errorf("Error building target %s: %s\nENVIRONMENT:\n%s\n%s\n%s",
				target.Label, err, strings.Join(cmd.Env, "\n"), target.GetCommand(), out)
		} else {
			return fmt.Errorf("Error building target %s: %s\n%s", target.Label, err, out)
		}
	}
	if target.PostBuildFunction != 0 {
		if err := parse.RunPostBuildFunction(tid, state, target, string(out)); err != nil {
			return err
		}
		storePostBuildOutput(state, target, out)
	}
	checkLicences(state, target)
	state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Collecting outputs...")
	outputsChanged, err := moveOutputs(state, target)
	if err != nil {
		return fmt.Errorf("Error moving outputs for target %s: %s", target.Label, err)
	} else if outputsChanged {
		target.SetState(core.Built)
	} else {
		target.SetState(core.Unchanged)
	}
	if state.Cache != nil && !target.SkipCache {
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Storing...")
		(*state.Cache).Store(target, mustShortTargetHash(state, target))
		if target.PostBuildFunction != 0 {
			// NB. Important this is stored with the earlier hash - if we calculate the hash
			//     now, it might be different, and we could of course never retrieve it again.
			(*state.Cache).StoreExtra(target, cacheKey, core.PostBuildOutputFileName(target))
		}
	}
	// Clean up the temporary directory once it's done.
	if state.CleanWorkdirs {
		if err := os.RemoveAll(target.TmpDir()); err != nil {
			log.Warning("Failed to remove temporary directory for %s: %s", target.Label, err)
		}
	}
	state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
	return nil
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
	return os.MkdirAll(directory, core.DirPermissions) // drwxrwxr-x
}

// Symlinks the source files of this rule into its temp directory.
func prepareSources(graph *core.BuildGraph, target *core.BuildTarget, dependency *core.BuildTarget, done map[core.BuildLabel]bool) error {
	for source := range core.IterSources(graph, target) {
		if err := core.PrepareSourcePair(source); err != nil {
			return err
		}
	}
	return nil
}

func moveOutputs(state *core.BuildState, target *core.BuildTarget) (bool, error) {
	// Before we write any outputs, we must remove the old hash file to avoid it being
	// left in an inconsistent state.
	if err := os.Remove(ruleHashFileName(target)); err != nil && !os.IsNotExist(err) {
		return true, err
	}
	for _, output := range target.Outputs() {
		tmpOutput := path.Join(target.TmpDir(), output)
		realOutput := path.Join(target.OutDir(), output)
		if !core.PathExists(tmpOutput) {
			return true, fmt.Errorf("Rule %s failed to create output %s", target.Label, tmpOutput)
		}
		// If output is a symlink, dereference it. Otherwise, for efficiency,
		// we can just move it without a full copy (saves copying large .jar files etc).
		dereferencedPath, err := filepath.EvalSymlinks(tmpOutput)
		if err != nil {
			return true, err
		}
		// hash the file
		newHash, err := pathHash(dereferencedPath)
		if err != nil {
			return true, err
		}
		// The tmp output can be a symlink back to the real one; this is allowed for rules like
		// filegroups that attempt to link outputs of other rules. In that case we can't
		// remove the original because that'd break the link, but by definition we don't need
		// to actually do anything more.
		// TODO(pebers): The logic here is quite tortured, consider a (very careful) rewrite.
		dereferencedOutput, _ := filepath.EvalSymlinks(realOutput)
		if absOutput, _ := filepath.Abs(realOutput); dereferencedPath == absOutput || realOutput == dereferencedPath || dereferencedOutput == dereferencedPath {
			continue
		}
		if core.PathExists(realOutput) {
			oldHash, err := pathHash(realOutput)
			if err != nil {
				return true, err
			} else if bytes.Equal(oldHash, newHash) {
				// We already have the same file in the current location. Don't bother moving it.
				continue
			}
		}
		if err := os.RemoveAll(realOutput); err != nil {
			return true, err
		}
		movePathHash(dereferencedPath, realOutput)
		// Check if we need a directory for this output.
		dir := path.Dir(realOutput)
		if !core.PathExists(dir) {
			if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
				return true, err
			}
		}
		// If the output file is in plz-out/tmp we can just move it to save time, otherwise we need
		// to copy so we don't move files from other directories.
		if strings.HasPrefix(dereferencedPath, target.TmpDir()) {
			if err := os.Rename(dereferencedPath, realOutput); err != nil {
				return true, err
			}
		} else {
			if err := core.RecursiveCopyFile(dereferencedPath, realOutput, 0, false); err != nil {
				return true, err
			}
		}
		if target.IsBinary {
			if err := os.Chmod(realOutput, 0775); err != nil {
				return true, err
			}
		}
	}
	return calculateAndCheckRuleHash(state, target), nil
}

// Removes all generated outputs for a rule.
func removeOutputs(target *core.BuildTarget) error {
	if err := os.Remove(ruleHashFileName(target)); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, output := range target.Outputs() {
		if err := os.RemoveAll(path.Join(target.OutDir(), output)); err != nil {
			return err
		}
	}
	return nil
}

// Checks the hash for a rule and compares it to any previous one.
// Returns true if outputs have changed from what was there previously.
func calculateAndCheckRuleHash(state *core.BuildState, target *core.BuildTarget) bool {
	hash, err := OutputHash(target)
	if err != nil {
		panic(err)
	}
	if err = checkRuleHashes(target, hash); err != nil {
		if state.NeedHashesOnly && state.IsOriginalTarget(target.Label) {
			panic(stopTarget("Hash mismatch"))
		} else if state.VerifyHashes {
			panic(err)
		} else {
			log.Warning("%s", err)
		}
	}
	// TODO(pebers): We should be able to get away without reading & writing this file quite so often...
	_, _, oldSourceHash := readRuleHashFile(ruleHashFileName(target), true)
	if err := writeRuleHashFile(state, target); err != nil {
		panic(fmt.Errorf("Attempting to create hash file: %s", err))
	}
	return !bytes.Equal(oldSourceHash, hash)
}

// OutputHash calculates the hash of a target's outputs.
func OutputHash(target *core.BuildTarget) ([]byte, error) {
	h := sha1.New()
	for _, output := range target.Outputs() {
		if h2, err := pathHash(path.Join(target.OutDir(), output)); err != nil {
			return nil, err
		} else {
			h.Write(h2)
		}
	}
	return h.Sum(nil), nil
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
	return fmt.Errorf("Bad output hash for rule %s: was %s, expected one of [%s]",
		target.Label, hashStr, strings.Join(target.Hashes, ", "))
}

func retrieveFromCache(state *core.BuildState, target *core.BuildTarget) ([]byte, bool) {
	hash := mustShortTargetHash(state, target)
	return hash, (*state.Cache).Retrieve(target, hash)
}

// Runs the post-build function for a target if it's got one.
func runPostBuildFunctionIfNeeded(tid int, state *core.BuildState, target *core.BuildTarget) {
	if target.PostBuildFunction != 0 {
		out := loadPostBuildOutput(state, target)
		if err := parse.RunPostBuildFunction(tid, state, target, out); err != nil {
			panic(err)
		}
	}
}

// checkLicences checks the licences for the target match what we've accepted / rejected in the config
// and panics if they don't match.
func checkLicences(state *core.BuildState, target *core.BuildTarget) {
	for _, licence := range target.Licences {
		for _, reject := range state.Config.Licences.Reject {
			if reject == licence {
				panic(fmt.Sprintf("Target %s is licensed %s, which is explicitly rejected for this repository", target.Label, licence))
			}
		}
		for _, accept := range state.Config.Licences.Accept {
			if accept == licence {
				log.Info("Licence %s is accepted in this repository", licence)
				return // Note licences are assumed to be an 'or', ie. any one of them can be accepted.
			}
		}
	}
	if len(target.Licences) > 0 && len(state.Config.Licences.Accept) > 0 {
		panic(fmt.Sprintf("None of the licences for %s are accepted in this repository: %s", target.Label, strings.Join(target.Licences, ", ")))
	}
}
