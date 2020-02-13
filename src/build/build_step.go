// Package build houses the core functionality for actually building targets.
package build

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/shlex"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/worker"
)

var log = logging.MustGetLogger("build")

// Type that indicates that we're stopping the build of a target in a nonfatal way.
var errStop = fmt.Errorf("stopping build")

// httpClient is the shared http client that we use for fetching remote files.
var httpClient http.Client
var httpClientOnce sync.Once

// Build implements the core logic for building a single target.
func Build(tid int, state *core.BuildState, label core.BuildLabel, remote bool) {
	target := state.Graph.TargetOrDie(label)
	state = state.ForTarget(target)
	target.SetState(core.Building)
	if err := buildTarget(tid, state, target, remote); err != nil {
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

	// Add any of the reverse deps that are now fully built to the queue.
	for _, reverseDep := range state.Graph.ReverseDependencies(target) {
		if reverseDep.State() == core.Active && state.Graph.AllDepsBuilt(reverseDep) && reverseDep.SyncUpdateState(core.Active, core.Pending) {
			state.AddPendingBuild(reverseDep.Label, false)
		}
	}
	if target.IsTest && state.NeedTests && ((state.IsOriginalTarget(target.Label) && state.ShouldInclude(target)) || state.IsExactOriginalTarget(target.Label)) {
		state.AddPendingTest(target.Label)
	}
}

// Builds a single target
func buildTarget(tid int, state *core.BuildState, target *core.BuildTarget, runRemotely bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%s", r)
			}
			log.Debug("Build failed: %s", err)
			log.Debug(string(debug.Stack()))
		}
	}()

	metadata := &core.BuildMetadata{StartTime: time.Now()}

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
			return fmt.Errorf("can't prepare temporary directory for %s; filegroups don't have temporary directories", target.Label)
		}
		if err := prepareDirectories(target); err != nil {
			return err
		}
		if err := prepareSources(state.Graph, target); err != nil {
			return err
		}
		// This is important to catch errors here where we will recover the panic, rather
		// than later when we shell into the temp dir.
		mustShortTargetHash(state, target)
		return errStop
	}
	var cacheKey, out []byte
	if runRemotely {
		m, err := state.RemoteClient.Build(tid, target)
		if err != nil {
			return err
		}
		out = m.Stdout
	} else {
		if target.IsHashFilegroup {
			updateHashFilegroupPaths(state, target)
		}
		// Ensure we have downloaded any previous dependencies if that's relevant.
		if state.Config.NumRemoteExecutors() > 0 {
			for _, dep := range target.Dependencies() {
				if dep.State() == core.BuiltRemotely {
					if err := state.RemoteClient.Download(dep); err != nil {
						return err
					}
				}
			}
		}

		// We don't record rule hashes for filegroups since we know the implementation and the check
		// is just "are these the same file" which we do anyway, and it means we don't have to worry
		// about two rules outputting the same file.
		haveRunPostBuildFunction := false
		if !target.IsFilegroup && !needsBuilding(state, target, false) {
			log.Debug("Not rebuilding %s, nothing's changed", target.Label)
			if target.PostBuildFunction != nil {
				postBuildOutput, err := loadPostBuildOutput(target)
				if err != nil {
					log.Warning("Missing post-build output for %s; will rebuild.", target.Label)
				} else if err := runPostBuildFunction(tid, state, target, postBuildOutput, ""); err != nil {
					log.Warning("Error from post-build function for %s: %s; will rebuild", target.Label, err)
				}
			}
			// If a post-build function ran it may modify the rule definition. In that case we
			// need to check again whether the rule needs building.
			if target.PostBuildFunction == nil || !needsBuilding(state, target, true) {
				if target.IsFilegroup {
					// Small optimisation to ensure we don't need to rehash things unnecessarily.
					copyFilegroupHashes(state, target)
				}
				target.SetState(core.Reused)
				state.LogBuildResult(tid, target.Label, core.TargetCached, "Unchanged")
				buildLinks(state, target)
				return nil // Nothing needs to be done.
			}
			log.Debug("Rebuilding %s after post-build function", target.Label)
			haveRunPostBuildFunction = true
		}
		if target.IsFilegroup {
			log.Debug("Building %s...", target.Label)
			if changed, err := buildFilegroup(state, target); err != nil {
				return err
			} else if _, err := calculateAndCheckRuleHash(state, target); err != nil {
				return err
			} else if changed {
				target.SetState(core.Built)
				state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
			} else {
				target.SetState(core.Unchanged)
				state.LogBuildResult(tid, target.Label, core.TargetCached, "Unchanged")
			}
			buildLinks(state, target)
			return nil
		}
		if err := prepareDirectories(target); err != nil {
			return fmt.Errorf("Error preparing directories for %s: %s", target.Label, err)
		}

		// N.B. Important we do not go through state.TargetHasher here since it memoises and
		//      this calculation might be incorrect.
		oldOutputHash, outputHashErr := outputHash(target, target.FullOutputs(), state.PathHasher, state.PathHasher.NewHash)
		retrieveArtifacts := func() bool {
			// If there aren't any outputs, we don't have to do anything right now.
			// Checks later will handle the case of something with a post-build function that
			// later tries to add more outputs.
			if len(target.DeclaredOutputs()) == 0 && len(target.DeclaredNamedOutputs()) == 0 {
				target.SetState(core.Unchanged)
				state.LogBuildResult(tid, target.Label, core.TargetCached, "Nothing to do")
				return true
			}
			state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Checking cache...")

			if state.Cache.Retrieve(target, mustShortTargetHash(state, target), target.Outputs()) != nil {
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
				buildLinks(state, target)
				return true // got from cache
			}
			log.Debug("Nothing retrieved from remote cache for %s", target.Label)
			return false
		}
		cacheKey = mustShortTargetHash(state, target)
		if state.Cache != nil && !runRemotely {
			// Note that ordering here is quite sensitive since the post-build function can modify
			// what we would retrieve from the cache.
			if target.PostBuildFunction != nil && !haveRunPostBuildFunction {
				log.Debug("Checking for post-build output for %s in cache...", target.Label)
				if metadata := state.Cache.Retrieve(target, cacheKey, nil); metadata != nil {
					storePostBuildOutput(target, metadata.Stdout)
					postBuildOutput = string(metadata.Stdout)
					if err := runPostBuildFunction(tid, state, target, postBuildOutput, ""); err != nil {
						return err
					} else if retrieveArtifacts() {
						return writeRuleHash(state, target)
					}
				}
			} else if retrieveArtifacts() {
				return nil
			}
		}
		if err := target.CheckSecrets(); err != nil {
			return err
		}
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Preparing...")
		if err := prepareSources(state.Graph, target); err != nil {
			return fmt.Errorf("Error preparing sources for %s: %s", target.Label, err)
		}

		state.LogBuildResult(tid, target.Label, core.TargetBuilding, target.BuildingDescription)
		out, err = buildMaybeRemotely(state, target, cacheKey)
		if err != nil {
			return err
		}
	}
	if target.PostBuildFunction != nil {
		out = bytes.TrimSpace(out)
		outs := target.Outputs()
		if err := runPostBuildFunction(tid, state, target, string(out), postBuildOutput); err != nil {
			return err
		}
		metadata.Stdout = out
		if !runRemotely {
			storePostBuildOutput(target, out)
		}
		if runRemotely && len(outs) != len(target.Outputs()) {
			// postBuildFunction has changed the target - must rebuild it
			log.Info("Rebuilding %s after post-build function", target)
			m, err := state.RemoteClient.Build(tid, target)
			if err != nil {
				return err
			}
			out = m.Stdout
		}
	}

	metadata.EndTime = time.Now()
	checkLicences(state, target)

	if runRemotely {
		target.SetState(core.BuiltRemotely)
		state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built remotely")
		return nil
	}

	state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Collecting outputs...")
	outs, outputsChanged, err := moveOutputs(state, target)
	if err != nil {
		return fmt.Errorf("error moving outputs for target %s: %s", target.Label, err)
	}
	if _, err = calculateAndCheckRuleHash(state, target); err != nil {
		return err
	}
	if outputsChanged {
		target.SetState(core.Built)
	} else {
		target.SetState(core.Unchanged)
	}
	buildLinks(state, target)
	if state.Cache != nil {
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Storing...")
		newCacheKey := mustShortTargetHash(state, target)
		if target.PostBuildFunction != nil {
			if !bytes.Equal(newCacheKey, cacheKey) {
				// NB. Important this is stored with the earlier hash - if we calculate the hash
				//     now, it might be different, and we could of course never retrieve it again.
				state.Cache.Store(target, cacheKey, metadata, nil)
			}
		}
		state.Cache.Store(target, newCacheKey, metadata, outs)
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
	env := core.StampedBuildEnvironment(state, target, inputHash, path.Join(core.RepoRoot, target.TmpDir()))
	log.Debug("Building target %s\nENVIRONMENT:\n%s\n%s", target.Label, env, command)
	out, combined, err := state.ProcessExecutor.ExecWithTimeoutShell(target, target.TmpDir(), env, target.BuildTimeout, state.ShowAllOutput, command, target.Sandbox)
	if err != nil {
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
	for source := range core.IterSources(graph, target, false) {
		if err := core.PrepareSourcePair(source); err != nil {
			return err
		}
	}
	if target.Stamp {
		if err := fs.WriteFile(bytes.NewReader(core.StampFile(target)), path.Join(target.TmpDir(), target.StampFileName()), 0644); err != nil {
			return err
		}
	}
	return nil
}

func moveOutputs(state *core.BuildState, target *core.BuildTarget) ([]string, bool, error) {
	changed := false
	tmpDir := target.TmpDir()
	outDir := target.OutDir()
	outs := target.Outputs()
	allOuts := make([]string, len(outs))
	for i, output := range outs {
		allOuts[i] = output
		tmpOutput := path.Join(tmpDir, target.GetTmpOutput(output))
		realOutput := path.Join(outDir, output)
		if !core.PathExists(tmpOutput) {
			return nil, true, fmt.Errorf("rule %s failed to create output %s", target.Label, tmpOutput)
		}
		outputChanged, err := moveOutput(state, target, tmpOutput, realOutput)
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
	for _, output := range fs.Glob(state.Config.Parse.BuildFileName, tmpDir, target.OptionalOutputs, nil, nil, true) {
		log.Debug("Discovered optional output %s", output)
		tmpOutput := path.Join(tmpDir, output)
		realOutput := path.Join(outDir, output)
		if _, err := moveOutput(state, target, tmpOutput, realOutput); err != nil {
			return nil, changed, err
		}
		allOuts = append(allOuts, output)
	}
	return allOuts, changed, nil
}

func moveOutput(state *core.BuildState, target *core.BuildTarget, tmpOutput, realOutput string) (bool, error) {
	// hash the file
	newHash, err := state.PathHasher.Hash(tmpOutput, false, true)
	if err != nil {
		return true, err
	}
	if fs.PathExists(realOutput) {
		if oldHash, err := state.PathHasher.Hash(realOutput, false, true); err != nil {
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
	state.PathHasher.MoveHash(tmpOutput, realOutput, false)
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
		if err := fs.RecursiveCopy(tmpOutput, realOutput, target.OutMode()); err != nil {
			return true, err
		}
	}
	return true, nil
}

// RemoveOutputs removes all generated outputs for a rule.
func RemoveOutputs(target *core.BuildTarget) error {
	for _, output := range target.Outputs() {
		out := path.Join(target.OutDir(), output)
		if err := os.RemoveAll(out); err != nil {
			return err
		} else if err := fs.EnsureDir(out); err != nil {
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
		for dir := path.Dir(filename); dir != "." && dir != "/" && path.Base(dir) != "plz-out"; dir = path.Dir(dir) {
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
	hash, err := state.TargetHasher.OutputHash(target)
	if err != nil {
		return nil, err
	}
	if err = checkRuleHashes(state, target, hash); err != nil {
		if state.NeedHashesOnly && (state.IsOriginalTarget(target.Label) || state.IsOriginalTarget(target.Label.Parent())) {
			return nil, errStop
		} else if state.VerifyHashes {
			return nil, err
		} else {
			log.Warning("%s", err)
		}
	}
	if !target.IsFilegroup {
		if err := writeRuleHash(state, target); err != nil {
			return nil, fmt.Errorf("Attempting to record rule hash: %s", err)
		}
	}
	// Set appropriate permissions on outputs
	if target.IsBinary {
		for _, output := range target.FullOutputs() {
			// Walk through the output,
			// if the output is a directory,apply output mode to the file instead of the directory
			err := fs.Walk(output, func(path string, isDir bool) error {
				if isDir {
					return nil
				}
				return os.Chmod(path, target.OutMode())
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return hash, nil
}

// A targetHasher is an implementation of the interface in core.
type targetHasher struct {
	State  *core.BuildState
	hashes map[*core.BuildTarget][]byte
	mutex  sync.RWMutex
}

// newTargetHasher returns a new TargetHasher
func newTargetHasher(state *core.BuildState) core.TargetHasher {
	return &targetHasher{
		State:  state,
		hashes: map[*core.BuildTarget][]byte{},
	}
}

// OutputHash calculates the standard output hash of a build target.
func (h *targetHasher) OutputHash(target *core.BuildTarget) ([]byte, error) {
	h.mutex.RLock()
	hash, present := h.hashes[target]
	h.mutex.RUnlock()
	if present {
		return hash, nil
	}
	hash, err := h.outputHash(target)
	if err != nil {
		return hash, err
	}
	h.SetHash(target, hash)
	return hash, nil
}

// SetHash sets a hash for a build target.
func (h *targetHasher) SetHash(target *core.BuildTarget, hash []byte) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.hashes[target] = hash
}

// outputHash calculates the output hash for a target, choosing an appropriate strategy.
func (h *targetHasher) outputHash(target *core.BuildTarget) ([]byte, error) {
	outs := target.FullOutputs()
	// For compatibility, leave this on the default for SHA1 hashing.
	if len(outs) == 1 && fs.FileExists(outs[0]) && h.State.Config.Build.HashFunction != "sha1" {
		return outputHash(target, outs, h.State.PathHasher, nil)
	}
	return outputHash(target, outs, h.State.PathHasher, h.State.PathHasher.NewHash)
}

// outputHash is a more general form of OutputHash that allows different hashing strategies.
// For example, one could choose to use sha256 instead of our usual sha1.
func outputHash(target *core.BuildTarget, outputs []string, hasher *fs.PathHasher, combine func() hash.Hash) ([]byte, error) {
	if combine == nil {
		// Must be a single output, just hash that directly.
		return hasher.Hash(outputs[0], true, !target.IsFilegroup)
	}
	h := combine()
	for _, filename := range outputs {
		// NB. Always force a recalculation of the output hashes here. Memoisation is not
		//     useful because by definition we are rebuilding a target, and can actively hurt
		//     in cases where we compare the retrieved cache artifacts with what was there before.
		h2, err := hasher.Hash(filename, true, !target.IsFilegroup)
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

func mustOutputHash(target *core.BuildTarget, outputs []string, hasher *fs.PathHasher, combine func() hash.Hash) []byte {
	h, _ := outputHash(target, outputs, hasher, combine)
	return h
}

// Verify the hash of output files for a rule match the ones set on it.
func checkRuleHashes(state *core.BuildState, target *core.BuildTarget, hash []byte) error {
	// TODO(peterebden): can we genericise all of this to lean on the hashers we're getting
	//                   from the state rather than so much hardcoding?
	const sha1Len = 2 * sha1.Size // x2 because of hex-encoding
	const sha256Len = 2 * sha256.Size
	if len(target.Hashes) == 0 {
		return nil // nothing to check
	}
	outputs := target.FullOutputs()
	if checkRuleHashesOfType(target, outputs, state.Hasher("sha1"), sha1.New, sha1Len) ||
		(len(outputs) == 1 && checkRuleHashesOfType(target, outputs, state.Hasher("sha1"), nil, sha1Len)) ||
		(len(outputs) != 1 && checkRuleHashesOfType(target, outputs, state.Hasher("sha256"), sha256.New, sha256Len)) ||
		(len(outputs) == 1 && checkRuleHashesOfType(target, outputs, state.Hasher("sha256"), nil, sha256Len)) {
		return nil
	}
	hashStr := hex.EncodeToString(hash)
	if len(target.Hashes) == 1 {
		return fmt.Errorf("Bad output hash for rule %s: was %s but expected %s",
			target.Label, hashStr, target.Hashes[0])
	}
	return fmt.Errorf("Bad output hash for rule %s: was %s but expected one of [%s]",
		target.Label, hashStr, strings.Join(target.Hashes, ", "))
}

// checkRuleHashesOfType checks any hashes on this rule of a single type.
// Currently we support SHA-1 and SHA-256, and also have specialisations for the case
// where a target has a single output so as not to double-hash it.
// It is a bit fiddly, but is organised this way to avoid calculating hashes of
// unused types unnecessarily since that could get quite expensive.
func checkRuleHashesOfType(target *core.BuildTarget, outputs []string, hasher *fs.PathHasher, combine func() hash.Hash, size int) bool {
	for _, h := range target.Hashes {
		// Hashes can have an arbitrary label prefix. Strip it off if present.
		if index := strings.LastIndexByte(h, ':'); index != -1 {
			h = strings.TrimSpace(h[index+1:])
		}
		if len(h) == size { // Check if the hash is of the right algorithm
			bhash, _ := outputHash(target, outputs, hasher, combine)
			if hex.EncodeToString(bhash) == h {
				return true
			}
		}
	}
	return false
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

// buildLinks builds links from the given target if it's labelled appropriately.
// For example, Go targets may link themselves into plz-out/go/src etc.
func buildLinks(state *core.BuildState, target *core.BuildTarget) {
	buildLinksOfType(state, target, "link:", os.Symlink)
	buildLinksOfType(state, target, "hlink:", os.Link)
}

type linkFunc func(string, string) error

func buildLinksOfType(state *core.BuildState, target *core.BuildTarget, prefix string, f linkFunc) {
	if labels := target.PrefixedLabels(prefix); len(labels) > 0 {
		env := core.BuildEnvironment(state, target, path.Join(core.RepoRoot, target.TmpDir()))
		for _, dest := range labels {
			destDir := path.Join(core.RepoRoot, os.Expand(dest, env.ReplaceEnvironment))
			srcDir := path.Join(core.RepoRoot, target.OutDir())
			for _, out := range target.Outputs() {
				linkIfNotExists(path.Join(srcDir, out), path.Join(destDir, out), f)
			}
		}
	}
}

// linkIfNotExists creates dest as a link to src if it doesn't already exist.
func linkIfNotExists(src, dest string, f linkFunc) {
	if !fs.PathExists(dest) {
		if err := fs.EnsureDir(dest); err != nil {
			log.Warning("Failed to create directory for %s: %s", dest, err)
		} else if err := f(src, dest); err != nil && !os.IsExist(err) {
			log.Warning("Failed to create %s: %s", dest, err)
		}
	}
}

// fetchRemoteFile fetches a remote file from a URL.
// This is a builtin for better efficiency and more control over the whole process.
func fetchRemoteFile(state *core.BuildState, target *core.BuildTarget) error {
	httpClientOnce.Do(func() {
		if state.Config.Build.HTTPProxy != "" {
			httpClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(state.Config.Build.HTTPProxy.AsURL()),
			}
		}
	})
	if err := prepareDirectory(target.OutDir(), false); err != nil {
		return err
	} else if err := prepareDirectory(target.TmpDir(), false); err != nil {
		return err
	}
	httpClient.Timeout = time.Duration(state.Config.Build.Timeout) // Can't set this when we init the client because config isn't loaded then.
	var err error
	for _, src := range target.Sources {
		if e := fetchOneRemoteFile(state, target, src.String()); e != nil {
			err = multierror.Append(err, e)
		} else {
			return nil
		}
	}
	return err
}

func fetchOneRemoteFile(state *core.BuildState, target *core.BuildTarget, url string) error {
	env := core.BuildEnvironment(state, target, path.Join(core.RepoRoot, target.TmpDir()))
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
	state.PathHasher.SetHash(tmpPath, h.Sum(nil))
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

// buildMaybeRemotely builds a target, either sending it to a remote worker if needed,
// or locally if not.
func buildMaybeRemotely(state *core.BuildState, target *core.BuildTarget, inputHash []byte) ([]byte, error) {
	workerCmd, workerArgs, localCmd, err := core.WorkerCommandAndArgs(state, target)
	if err != nil {
		return nil, err
	} else if workerCmd == "" {
		return runBuildCommand(state, target, localCmd, inputHash)
	}
	// The scheme here is pretty minimal; remote workers currently have quite a bit less info than
	// local ones get. Over time we'll probably evolve it to add more information.
	opts, err := shlex.Split(workerArgs)
	if err != nil {
		return nil, err
	}
	log.Debug("Sending remote build request for %s to %s; opts %s", target.Label, workerCmd, workerArgs)
	resp, err := worker.BuildRemotely(state, target, workerCmd, &worker.Request{
		Rule:    target.Label.String(),
		Labels:  target.Labels,
		TempDir: path.Join(core.RepoRoot, target.TmpDir()),
		Sources: target.AllSourcePaths(state.Graph),
		Options: opts,
	})
	if err != nil {
		return nil, err
	}
	out := strings.Join(resp.Messages, "\n")
	if !resp.Success {
		return nil, fmt.Errorf("Error building target %s: %s", target.Label, out)
	}
	// Okay, now we might need to do something locally too...
	if localCmd != "" {
		out2, err := runBuildCommand(state, target, localCmd, inputHash)
		return append([]byte(out+"\n"), out2...), err
	}
	return []byte(out), nil
}
