// Package build houses the core functionality for actually building targets.
package build

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/op/go-logging.v1"

	"core"
	"metrics"
)

var log = logging.MustGetLogger("build")

// Type that indicates that we're stopping the build of a target in a nonfatal way.
var errStop = fmt.Errorf("stopping build")

// buildingFilegroupOutputs is used to track any file that's being built by a filegroup right now.
// This avoids race conditions with them where two filegroups can race to build the same file
// simultaneously (that's impossible with other targets because we prohibit two rules from
// outputting the same file, for this and other reasons).
var buildingFilegroupOutputs = map[string]*sync.Mutex{}
var buildingFilegroupMutex sync.Mutex

// goDirOnce guards the creation of plz-out/go, which we only attempt once per process.
var goDirOnce sync.Once

// Build implements the core logic for building a single target.
func Build(tid int, state *core.BuildState, label core.BuildLabel) {
	start := time.Now()
	target := state.Graph.TargetOrDie(label)
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

	if err := target.CheckDependencyVisibility(state.Graph); err != nil {
		return err
	}
	// We can't do this check until build time, until then we don't know what all the outputs
	// will be (eg. for filegroups that collect outputs of other rules).
	if err := target.CheckDuplicateOutputs(); err != nil {
		return err
	}
	// This must run before we can leave this function successfully by any path.
	if target.PreBuildFunction != 0 {
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
		if postBuildOutput, err = runPostBuildFunctionIfNeeded(tid, state, target); err != nil {
			log.Warning("Missing post-build output for %s; will rebuild.", target.Label)
		} else {
			// If a post-build function ran it may modify the rule definition. In that case we
			// need to check again whether the rule needs building.
			if target.PostBuildFunction == 0 || !needsBuilding(state, target, true) {
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
	if target.IsFilegroup {
		log.Debug("Building %s...", target.Label)
		return buildFilegroup(tid, state, target)
	}
	oldOutputHash, outputHashErr := OutputHash(target)
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
		if target.PostBuildFunction != 0 {
			log.Debug("Checking for post-build output file for %s in cache...", target.Label)
			if state.Cache.RetrieveExtra(target, cacheKey, target.PostBuildOutputFileName()) {
				if postBuildOutput, err = runPostBuildFunctionIfNeeded(tid, state, target); err != nil {
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
	if target.PostBuildFunction != 0 {
		out = bytes.TrimSpace(out)
		sout := string(out)
		if postBuildOutput != "" {
			// We've already run the post-build function once, it's not safe to do it again (e.g. if adding new
			// targets, it will likely fail). Theoretically it should get the same output this time and hence would
			// do the same thing, since it had all the same inputs.
			// Obviously we can't be 100% sure that will be the case, so issue a warning if not...
			if postBuildOutput != sout {
				log.Warning("The build output for %s differs from what we got back from the cache earlier.\n"+
					"This implies your target's output is nondeterministic; Please won't re-run the\n"+
					"post-build function, which will *probably* be okay, but Please can't be sure.\n"+
					"See https://github.com/thought-machine/please/issues/113 for more information.", target.Label)
				log.Debug("Cached build output for %s: %s\n\nNew build output: %s",
					target.Label, postBuildOutput, sout)
			}
		} else if err := state.Parser.RunPostBuildFunction(tid, state, target, sout); err != nil {
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
		if target.PostBuildFunction != 0 {
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
	env := core.StampedBuildEnvironment(state, target, false, inputHash)
	log.Debug("Building target %s\nENVIRONMENT:\n%s\n%s", target.Label, env, command)
	out, combined, err := core.ExecWithTimeoutShell(target, target.TmpDir(), env, target.BuildTimeout, state.Config.Build.Timeout, state.ShowAllOutput, command, target.Sandbox)
	if err != nil {
		if state.Verbosity >= 4 {
			return nil, fmt.Errorf("Error building target %s: %s\nENVIRONMENT:\n%s\n%s\n%s",
				target.Label, err, env, target.GetCommand(), combined)
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
	return os.MkdirAll(directory, core.DirPermissions) // drwxrwxr-x
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
		// If output is a symlink, dereference it. Otherwise, for efficiency,
		// we can just move it without a full copy (saves copying large .jar files etc).
		dereferencedPath, err := filepath.EvalSymlinks(tmpOutput)
		if err != nil {
			return nil, true, err
		}
		// NB. false -> not filegroup, we wouldn't be here if it was.
		outputChanged, err := moveOutput(target, dereferencedPath, realOutput, false)
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
	for _, output := range core.Glob(tmpDir, target.OptionalOutputs, nil, nil, true) {
		log.Debug("Discovered optional output %s", output)
		tmpOutput := path.Join(tmpDir, output)
		realOutput := path.Join(outDir, output)
		if _, err := moveOutput(target, tmpOutput, realOutput, false); err != nil {
			return nil, changed, err
		}
		extraOuts = append(extraOuts, output)
	}
	return extraOuts, changed, nil
}

func moveOutput(target *core.BuildTarget, tmpOutput, realOutput string, filegroup bool) (bool, error) {
	// hash the file
	newHash, err := pathHash(tmpOutput, false)
	if err != nil {
		return true, err
	}
	realOutputExists := core.PathExists(realOutput)
	// If this is a filegroup we hardlink the outputs over and so the two files may actually be
	// the same file. If so don't do anything else and especially don't delete & recreate the
	// file because other things might be using it already (because more than one filegroup can
	// own the same file).
	if filegroup && realOutputExists && core.IsSameFile(tmpOutput, realOutput) {
		log.Debug("real output %s is same file", realOutput)
		movePathHash(tmpOutput, realOutput, filegroup) // make sure this is updated regardless
		return false, nil
	}
	if realOutputExists {
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
	movePathHash(tmpOutput, realOutput, filegroup)
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
		if err := core.RecursiveCopyFile(tmpOutput, realOutput, target.OutMode(), filegroup, false); err != nil {
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
		return err
	}
	for _, output := range target.Outputs() {
		if err := os.RemoveAll(path.Join(target.OutDir(), output)); err != nil {
			return err
		}
	}
	return nil
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
func runPostBuildFunctionIfNeeded(tid int, state *core.BuildState, target *core.BuildTarget) (string, error) {
	if target.PostBuildFunction != 0 {
		out, err := loadPostBuildOutput(state, target)
		if err != nil {
			return "", err
		}
		if err := state.Parser.RunPostBuildFunction(tid, state, target, out); err != nil {
			panic(err)
		}
		return out, nil
	}
	return "", nil
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

// buildFilegroup runs the manual build steps for a filegroup rule.
// We don't force this to be done in bash to avoid errors with maximum command lengths,
// and it's actually quite fiddly to get just so there.
func buildFilegroup(tid int, state *core.BuildState, target *core.BuildTarget) error {
	if err := prepareDirectory(target.OutDir(), false); err != nil {
		return err
	}
	if err := os.RemoveAll(ruleHashFileName(target)); err != nil {
		return err
	}
	changed := false
	outDir := target.OutDir()
	localSources := target.AllLocalSourcePaths(state.Graph)
	for i, source := range target.AllFullSourcePaths(state.Graph) {
		out, _ := filegroupOutputPath(target, outDir, localSources[i], source)
		c, err := buildFilegroupFile(target, source, out)
		if err != nil {
			return err
		}
		changed = changed || c
	}
	if target.HasLabel("py") && !target.IsBinary {
		// Pre-emptively create __init__.py files so the outputs can be loaded dynamically.
		// It's a bit cheeky to do non-essential language-specific logic but this enables
		// a lot of relatively normal Python workflows.
		// Errors are deliberately ignored.
		if pkg := state.Graph.Package(target.Label.PackageName); pkg == nil || !pkg.HasOutput("__init__.py") {
			// Don't create this if someone else is going to create this in the package.
			createInitPy(outDir)
		}
	}
	if _, err := calculateAndCheckRuleHash(state, target); err != nil {
		return err
	} else if changed {
		target.SetState(core.Built)
	} else {
		target.SetState(core.Unchanged)
	}
	state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
	return nil
}

func buildFilegroupFile(target *core.BuildTarget, fromPath, toPath string) (bool, error) {
	buildingFilegroupMutex.Lock()
	m, present := buildingFilegroupOutputs[toPath]
	if !present {
		m = &sync.Mutex{}
		buildingFilegroupOutputs[toPath] = m
	}
	m.Lock()
	buildingFilegroupMutex.Unlock()
	changed, err := moveOutput(target, fromPath, toPath, true)
	m.Unlock()
	buildingFilegroupMutex.Lock()
	delete(buildingFilegroupOutputs, toPath)
	buildingFilegroupMutex.Unlock()
	return changed, err
}

// copyFilegroupHashes copies the hashes of the inputs of this filegroup to their outputs.
// This is a small optimisation to ensure we don't need to recalculate them unnecessarily.
func copyFilegroupHashes(state *core.BuildState, target *core.BuildTarget) {
	outDir := target.OutDir()
	localSources := target.AllLocalSourcePaths(state.Graph)
	for i, source := range target.AllFullSourcePaths(state.Graph) {
		if out, _ := filegroupOutputPath(target, outDir, localSources[i], source); out != source {
			movePathHash(source, out, true)
		}
	}
}

// updateHashFilegroupPaths sets the output paths on a hash_filegroup rule.
// Unlike normal filegroups, hash filegroups can't calculate these themselves very readily.
func updateHashFilegroupPaths(state *core.BuildState, target *core.BuildTarget) {
	outDir := target.OutDir()
	localSources := target.AllLocalSourcePaths(state.Graph)
	for i, source := range target.AllFullSourcePaths(state.Graph) {
		_, relOut := filegroupOutputPath(target, outDir, localSources[i], source)
		target.AddOutput(relOut)
	}
}

// filegroupOutputPath returns the output path for a single filegroup source.
func filegroupOutputPath(target *core.BuildTarget, outDir, source, full string) (string, string) {
	if !target.IsHashFilegroup {
		return path.Join(outDir, source), source
	}
	// Hash filegroups have a hash embedded into the output name.
	ext := path.Ext(source)
	before := source[:len(source)-len(ext)]
	hash, err := pathHash(full, false)
	if err != nil {
		panic(err)
	}
	out := before + "-" + base64.RawURLEncoding.EncodeToString(hash) + ext
	return path.Join(outDir, out), out
}

func createInitPy(dir string) {
	initPy := path.Join(dir, "__init__.py")
	if core.PathExists(initPy) {
		return
	}
	if f, err := os.OpenFile(initPy, os.O_RDONLY|os.O_CREATE, 0444); err == nil {
		f.Close()
	}
	dir = path.Dir(dir)
	if dir != core.GenDir && dir != "." && !core.PathExists(path.Join(dir, "__init__.py")) {
		createInitPy(dir)
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
