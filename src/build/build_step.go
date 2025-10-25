// Package build houses the core functionality for actually building targets.
package build

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/thought-machine/please/src/audit"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/generate"
	"github.com/thought-machine/please/src/metrics"
	"github.com/thought-machine/please/src/process"
	"github.com/thought-machine/please/src/version"
)

var log = logging.Log

// Type that indicates that we're stopping the build of a target in a nonfatal way.
var errStop = fmt.Errorf("stopping build")

// httpClient is the shared http client that we use for fetching remote files.
var httpClient *retryablehttp.Client
var httpClientOnce sync.Once
var httpClientLimiter chan struct{}

var successfulRemoteTargetBuildDuration = metrics.NewHistogramVec(
	"remote",
	"target_build_duration",
	"Time taken to successfully build a target, in milliseconds",
	metrics.ExponentialBuckets(0.5, 2, 16), // 16 buckets, starting at 0.5ms and doubling in width.
	[]string{"ci"},
)

var successfulLocalTargetBuildDuration = metrics.NewHistogramVec(
	"local",
	"target_build_duration",
	"Time taken to successfully build a target, in milliseconds",
	metrics.ExponentialBuckets(0.5, 2, 16), // 16 buckets, starting at 0.5ms and doubling in width.
	[]string{"ci"},
)

// Build implements the core logic for building a single target.
func Build(state *core.BuildState, target *core.BuildTarget, remote bool) {
	state = state.ForTarget(target)
	target.SetState(core.Building)
	start := time.Now()
	if err := buildTarget(state, target, remote); err != nil {
		if errors.Is(err, errStop) {
			target.SetState(core.Stopped)
			state.LogBuildResult(target, core.TargetBuildStopped, "Build stopped")
			return
		}
		state.LogBuildError(target.Label, core.TargetBuildFailed, err, "Build failed: %s", err)
		if err := RemoveOutputs(target); err != nil {
			log.Errorf("Failed to remove outputs for %s: %s", target.Label, err)
		}
		target.SetState(core.Failed)
		target.FinishBuild()
		return
	}
	if remote {
		successfulRemoteTargetBuildDuration.WithLabelValues(metrics.CILabel).Observe(float64(time.Since(start).Milliseconds()))
	} else {
		successfulLocalTargetBuildDuration.WithLabelValues(metrics.CILabel).Observe(float64(time.Since(start).Milliseconds()))
	}
	// Mark the target as having finished building.
	target.FinishBuild()
}

func validateBuildTargetBeforeBuild(state *core.BuildState, target *core.BuildTarget) error {
	if err := target.CheckDependencyVisibility(state); err != nil {
		return err
	}
	// We can't do this check until build time, until then we don't know what all the outputs
	// will be (eg. for filegroups that collect outputs of other rules).
	if err := target.CheckDuplicateOutputs(); err != nil {
		return err
	}

	// Check that the build inputs don't belong to another package
	if err := target.CheckTargetOwnsBuildInputs(state); err != nil {
		return err
	}

	// Check that the build outputs don't belong to another package
	return target.CheckTargetOwnsBuildOutputs(state)
}

func findFilegroupSourcesWithTmpDir(target *core.BuildTarget) []core.BuildLabel {
	srcs := make([]core.BuildLabel, 0, len(target.Sources))
	for _, src := range target.Sources {
		if l, ok := src.Label(); ok {
			srcs = append(srcs, l)
		}
	}
	return srcs
}

func prepareOnly(state *core.BuildState, target *core.BuildTarget) error {
	if target.IsFilegroup {
		potentialTargets := findFilegroupSourcesWithTmpDir(target)
		if len(potentialTargets) > 0 {
			return fmt.Errorf("can't prepare temporary directory for %s; filegroups don't have temporary directories... Perhaps you meant one of its srcs: %v", target.Label, potentialTargets)
		}

		return fmt.Errorf("can't prepare temporary directory for %s; filegroups don't have temporary directories", target.Label)
	}

	if state.RemoteClient != nil {
		// Targets were built remotely so we can simply download the inputs and place them in the
		// tmp/ folder and exit.
		state.LogBuildResult(target, core.TargetBuilding, "Downloading inputs...")
		if err := state.DownloadAllInputs(target, target.TmpDir(), false); err != nil {
			return err
		}

		return errStop
	}

	if err := prepareDirectories(target); err != nil {
		return err
	}
	if err := prepareSources(state, state.Graph, target); err != nil {
		return err
	}
	// This is important to catch errors here where we will recover the panic, rather
	// than later when we shell into the temp dir.
	mustShortTargetHash(state, target)
	return errStop
}

// Builds a single target
// This function takes the following steps:
//  1. Check if we have already built the rule
//     a) checks the hashes on xargs of all input files (rule, config, source, secret)
//     b) re-applies any updates that might have happened during build (the post-build and output dirs)
//     c) re-checks the hashes to see if those updates changed anything and need to re-build otherwise returns (nothing to do)
//  2. Checks if we have the build output cached
//     a) if the action of building this target could've changed how we calculate the output hash,
//     i)  attempt to fetch just the MD from the cache based on the old hashkey
//     ii) apply these updates to the outs based on the stored metadata (out dirs + run post build action)
//     b) attempt to fetch the outputs from the cache based on the output hash
//  3. Actually build the rule
//  4. Store result in the cache
func buildTarget(state *core.BuildState, target *core.BuildTarget, runRemotely bool) (err error) {
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

	err = validateBuildTargetBeforeBuild(state, target)
	if err != nil {
		return err
	}

	// This must run before we can leave this function successfully by any path.
	if target.PreBuildFunction != nil {
		log.Debug("Running pre-build function for %s", target.Label)
		if err := state.Parser.RunPreBuildFunction(state, target); err != nil {
			return err
		}
		log.Debug("Finished pre-build function for %s", target.Label)

		// Wait for any new dependencies added by pre-build commands before continuing.
		for _, dep := range target.Dependencies() {
			dep.WaitForBuild(target.Label)
			if dep.State() >= core.DependencyFailed { // Either the target failed or its dependencies failed
				// Give up and set the original target as dependency failed
				target.SetState(core.DependencyFailed)
				return fmt.Errorf("error in pre-rule dependency for %s: %s", target.Label, dep.Label)
			}
		}
		log.Debug("Finished waiting for dependencies for %s", target.Label)
	}

	state.LogBuildResult(target, core.TargetBuilding, "Preparing...")
	if state.PrepareOnly && state.IsOriginalTarget(target) && !state.NeedTests {
		return prepareOnly(state, target)
	}

	var postBuildOutput string
	var cacheKey []byte
	var metadata *core.BuildMetadata

	if target.HasLabel("go") {
		// Create a dummy go.mod file so Go tooling ignores the contents of plz-out.
		goModOnce.Do(writeGoMod)
	}

	if runRemotely {
		metadata, err = state.RemoteClient.Build(target)
		if err != nil {
			return err
		}
	} else {
		// Wait if another process is currently building this target
		state.LogBuildResult(target, core.TargetBuilding, "Acquiring target lock...")
		file := core.AcquireExclusiveFileLock(target.BuildLockFile())
		defer core.ReleaseFileLock(file)
		state.LogBuildResult(target, core.TargetBuilding, "Preparing...")

		// Ensure we have downloaded any previous dependencies if that's relevant.
		if err := state.DownloadInputsIfNeeded(target, false); err != nil {
			return err
		}

		// We don't record rule hashes for filegroups since we know the implementation and the check
		// is just "are these the same file" which we do anyway, and it means we don't have to worry
		// about two rules outputting the same file.
		haveRunPostBuildFunction := false
		if !target.IsFilegroup && !needsBuilding(state, target, false) {
			log.Debug("Not rebuilding %s, nothing's changed", target.Label)

			// If running the build could update the target,(e.g. via a post build action) we need to restore those
			// changes from the build metadata and check if we need to build the target again
			if target.BuildCouldModifyTarget() {
				// needsBuilding checks that the metadata file exists so this is safe
				metadata, err = loadTargetMetadata(target)
				if err != nil {
					return fmt.Errorf("failed to load build metadata for %s: %w", target.Label, err)
				}

				addOutDirOutsFromMetadata(target, metadata)

				if target.PostBuildFunction != nil {
					if err := runPostBuildFunction(state, target, string(metadata.Stdout), ""); err != nil {
						log.Warning("Error from post-build function for %s: %s; will rebuild", target.Label, err)
					}
				}
			}

			// If we still don't need to build, return immediately
			if !target.BuildCouldModifyTarget() || !needsBuilding(state, target, true) {
				if target.IsFilegroup {
					// Small optimisation to ensure we don't need to rehash things unnecessarily.
					copyFilegroupHashes(state, target)
				}
				target.SetState(core.Reused)
				state.LogBuildResult(target, core.TargetCached, "Unchanged")
				buildLinks(state, target)
				return nil // Nothing needs to be done.
			}
			log.Debug("Rebuilding %s after post-build function", target.Label)
			haveRunPostBuildFunction = true
		}
		if target.IsFilegroup {
			log.Debug("Building %s...", target.Label)
			changed, err := buildFilegroup(state, target)
			if err != nil {
				return err
			}
			if changed {
				if _, err := calculateAndCheckRuleHash(state, target); err != nil {
					return err
				}
				target.SetState(core.Built)
				state.LogBuildResult(target, core.TargetBuilt, "Built")
			} else {
				target.SetState(core.Unchanged)
				state.LogBuildResult(target, core.TargetCached, "Unchanged")
			}
			buildLinks(state, target)
			return nil
		}

		if err := prepareDirectories(target); err != nil {
			return fmt.Errorf("Error preparing directories for %s: %s", target.Label, err)
		}

		// If we fail to hash our outputs, we get a nil hash so we'll attempt to pull the outputs from the cache
		//
		// N.B. Important we do not go through state.TargetHasher here since it memoises and
		//      this calculation might be incorrect.
		oldOutputHash := outputHashOrNil(target, target.FullOutputs(), state.PathHasher, state.PathHasher.NewHash)
		cacheKey = mustShortTargetHash(state, target)

		if state.Cache != nil && !runRemotely && !state.ShouldRebuild(target) {
			// Note that ordering here is quite sensitive since the post-build function can modify
			// what we would retrieve from the cache.
			if target.BuildCouldModifyTarget() {
				log.Debug("Checking for build metadata for %s in cache...", target.Label)
				if metadata = retrieveFromCache(state.Cache, target, cacheKey, nil); metadata != nil {
					addOutDirOutsFromMetadata(target, metadata)
					if target.PostBuildFunction != nil && !haveRunPostBuildFunction {
						postBuildOutput = string(metadata.Stdout)
						if err := runPostBuildFunction(state, target, postBuildOutput, ""); err != nil {
							return err
						}
					}
					// Now that we've updated the rule, retrieve the artifacts with the new output hash
					if retrieveArtifacts(state, target, oldOutputHash) {
						return writeRuleHash(state, target)
					}
				}
			} else if retrieveArtifacts(state, target, oldOutputHash) {
				return nil
			}
		}
		if err := target.CheckSecrets(); err != nil {
			return err
		}
		state.LogBuildResult(target, core.TargetBuilding, "Preparing...")
		if err := prepareSources(state, state.Graph, target); err != nil {
			return fmt.Errorf("Error preparing sources for %s: %s", target.Label, err)
		}

		state.LogBuildResult(target, core.TargetBuilding, target.BuildingDescription)
		metadata, err = build(state, target, cacheKey)
		if err != nil {
			return err
		}

		// Add optional outputs to target metadata
		metadata.OptionalOutputs = make([]string, 0)
		for _, output := range fs.Glob(fs.HostFS, state.Config.Parse.BuildFileName, target.TmpDir(), target.OptionalOutputs, nil, true) {
			log.Debug("Add discovered optional output to metadata %s", output)
			metadata.OptionalOutputs = append(metadata.OptionalOutputs, output)
		}

		metadata.OutputDirOuts, err = addOutputDirectoriesToBuildOutput(target)
		if err != nil {
			return err
		}
	}

	if target.PostBuildFunction != nil {
		outs := target.Outputs()
		if err := runPostBuildFunction(state, target, string(metadata.Stdout), postBuildOutput); err != nil {
			return err
		}

		// Wait for any new dependencies added by post-build commands before continuing.
		for _, dep := range target.Dependencies() {
			dep.WaitForBuild(target.Label)
			if dep.State() >= core.DependencyFailed { // Either the target failed or its dependencies failed
				// Give up and set the original target as dependency failed
				target.SetState(core.DependencyFailed)
				return fmt.Errorf("error in post-rule dependency for %s: %s", target.Label, dep.Label)
			}
		}

		if runRemotely && len(outs) != len(target.Outputs()) {
			// postBuildFunction has changed the target - must rebuild it
			log.Info("Rebuilding %s after post-build function", target)
			metadata, err = state.RemoteClient.Build(target)
			if err != nil {
				return err
			}
		}
	}

	checkLicences(state, target)

	if runRemotely {
		if metadata.Cached {
			target.SetState(core.ReusedRemotely)
			state.LogBuildResult(target, core.TargetBuilt, "Reused existing action")
		} else {
			target.SetState(core.BuiltRemotely)
			state.LogBuildResult(target, core.TargetBuilt, "Built remotely")
		}
		if state.ShouldDownload(target) {
			if err := state.EnsureDownloaded(target); err != nil {
				return err
			}
			buildLinks(state, target)
		}
		return nil
	} else if err := StoreTargetMetadata(target, metadata); err != nil {
		return fmt.Errorf("failed to store target build metadata for %s: %w", target.Label, err)
	}

	state.LogBuildResult(target, core.TargetBuilding, "Collecting outputs...")
	outs, outputsChanged, err := moveOutputs(state, target)
	if err != nil {
		return fmt.Errorf("error moving outputs for target %s: %w", target.Label, err)
	}
	if _, err = calculateAndCheckRuleHash(state, target); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}
	if outputsChanged {
		target.SetState(core.Built)
	} else {
		target.SetState(core.Unchanged)
	}
	buildLinks(state, target)
	if state.Cache != nil {
		state.LogBuildResult(target, core.TargetBuilding, "Storing...")
		newCacheKey := mustShortTargetHash(state, target)

		// If the build could modify the target, store the metadata in the cache based on the original state of the
		// target. This is so it can be retrieved to apply the modifications so we can retrieve the files based on the
		// modified target hash.
		if target.BuildCouldModifyTarget() {
			if !bytes.Equal(newCacheKey, cacheKey) {
				// NB. Important this is stored with the earlier hash - if we calculate the hash
				//     now, it might be different, and we could of course never retrieve it again.
				storeInCache(state.Cache, target, cacheKey, nil)
			}
		}
		storeInCache(state.Cache, target, newCacheKey, outs)
	}
	// Clean up the temporary directory once it's done.
	if state.CleanWorkdirs {
		if err := fs.RemoveAll(target.TmpDir()); err != nil {
			log.Warning("Failed to remove temporary directory for %s: %s", target.Label, err)
		}
	}
	if outputsChanged {
		state.LogBuildResult(target, core.TargetBuilt, "Built")
	} else {
		state.LogBuildResult(target, core.TargetBuilt, "Built (unchanged)")
	}
	return nil
}

func outputHashOrNil(target *core.BuildTarget, outputs []string, hasher *fs.PathHasher, combine func() hash.Hash) []byte {
	h, err := outputHash(target, outputs, hasher, combine)
	if err != nil {
		// We might get an error because somebody deleted the outputs from plz-out. In this case return nil and attempt
		// to rebuild or fetch from the cache.
		return nil
	}
	return h
}

func addOutDirOutsFromMetadata(target *core.BuildTarget, md *core.BuildMetadata) {
	for _, o := range md.OutputDirOuts {
		target.AddOutput(o)
	}
}

func retrieveFromCache(cache core.Cache, target *core.BuildTarget, cacheKey []byte, files []string) *core.BuildMetadata {
	files = append(files, target.TargetBuildMetadataFileName())
	if ok := cache.Retrieve(target, cacheKey, files); ok {
		md, err := loadTargetMetadata(target)
		if err != nil {
			log.Debugf("failed to retrieve %s build metadata from cache: %v", target.Label, err)
			return nil
		}
		return md
	}
	return nil
}

func storeInCache(cache core.Cache, target *core.BuildTarget, key []byte, files []string) {
	files = append(files, target.TargetBuildMetadataFileName())
	cache.Store(target, key, files)
}

// retrieveArtifacts attempts to retrieve artifacts from the cache
//  1. if there are no declared outputs, return true; there's nothing to be done
//  2. pull all the declared outputs from the cache has based on the short hash of the target
//  3. check that pulling the artifacts changed the output hash and set the build state accordingly
func retrieveArtifacts(state *core.BuildState, target *core.BuildTarget, oldOutputHash []byte) bool {
	// If there aren't any outputs, we don't have to do anything right now.
	// Checks later will handle the case of something with a post-build function that
	// later tries to add more outputs.
	if len(target.DeclaredOutputs()) == 0 && len(target.DeclaredNamedOutputs()) == 0 {
		target.SetState(core.Unchanged)
		state.LogBuildResult(target, core.TargetCached, "Nothing to do")
		return true
	}
	state.LogBuildResult(target, core.TargetBuilding, "Checking cache...")

	cacheKey := mustShortTargetHash(state, target)

	if md := retrieveFromCache(state.Cache, target, cacheKey, target.Outputs()); md != nil {
		// Retrieve additional optional outputs from metadata
		if len(md.OptionalOutputs) > 0 {
			state.Cache.Retrieve(target, cacheKey, md.OptionalOutputs)
		}

		log.Debug("Retrieved artifacts for %s from cache", target.Label)
		checkLicences(state, target)
		newOutputHash, err := calculateAndCheckRuleHash(state, target)
		if err != nil { // Most likely hash verification failure
			log.Warning("Error retrieving cached artifacts for %s: %s", target.Label, err)
			RemoveOutputs(target)
			return false
		} else if oldOutputHash == nil || !bytes.Equal(oldOutputHash, newOutputHash) {
			target.SetState(core.Cached)
			state.LogBuildResult(target, core.TargetCached, "Cached")
		} else {
			target.SetState(core.Unchanged)
			state.LogBuildResult(target, core.TargetCached, "Cached (unchanged)")
		}
		buildLinks(state, target)

		// If we could've potentially pulled from the http cache, we need to write the xattrs back as they will be
		// missing.
		if state.Config.Cache.HTTPURL != "" {
			if err := writeRuleHash(state, target); err != nil {
				log.Warningf("failed to write target hash: %w", err)
				return false
			}
		}

		return true // got from cache
	}
	log.Debug("Nothing retrieved from remote cache for %s", target.Label)
	return false
}

// runBuildCommand runs the actual command to build a target.
// On success it returns the stdout of the target, otherwise an error.
func runBuildCommand(state *core.BuildState, target *core.BuildTarget, command string, inputHash []byte) ([]byte, error) {
	if target.IsRemoteFile {
		return nil, fetchRemoteFile(state, target)
	}
	if target.IsTextFile {
		return nil, buildTextFile(state, target)
	}
	env := core.StampedBuildEnvironment(state, target, inputHash, filepath.Join(core.RepoRoot, target.TmpDir()), target.Stamp).ToSlice()
	log.Debug("Building target %s\nENVIRONMENT:\n%s\n%s", target.Label, env, command)
	audit.WriteBuildCommand(target.Label.String(), env, command)
	out, combined, err := state.ProcessExecutor.ExecWithTimeoutShell(target, target.TmpDir(), env, target.BuildTimeout, state.ShowAllOutput, false, process.NewSandboxConfig(target.Sandbox, target.Sandbox), command)
	if err != nil {
		return nil, fmt.Errorf("Error building target %s: %s\n%s", target.Label, err, combined)
	}
	return out, nil
}

// buildTextFile runs the build action for text_file() rules
func buildTextFile(state *core.BuildState, target *core.BuildTarget) error {
	outs := target.Outputs()
	if len(outs) != 1 {
		return fmt.Errorf("text_file %s should have a single output, has %d", target.Label, len(outs))
	}
	outFile := filepath.Join(target.TmpDir(), outs[0])

	content, err := target.GetFileContent(state)
	if err != nil {
		return err
	}

	return os.WriteFile(outFile, []byte(content), target.OutMode())
}

// prepareOutputDirectories creates any directories the target has declared it will output into as a nicety
func prepareOutputDirectories(target *core.BuildTarget) error {
	for _, dir := range target.OutputDirectories {
		if err := prepareParentDirs(target, dir.Dir()); err != nil {
			return err
		}
	}

	for _, out := range target.Outputs() {
		if err := prepareParentDirs(target, out); err != nil {
			return err
		}
	}
	return nil
}

// prepareParentDirs will create any parent directories of an output i.e. for the output foo/bar/baz it will create
// foo and foo/bar
func prepareParentDirs(target *core.BuildTarget, out string) error {
	if dir := filepath.Dir(out); dir != "." {
		outPath := filepath.Join(target.TmpDir(), dir)
		if !core.PathExists(outPath) {
			if err := os.MkdirAll(outPath, core.DirPermissions); err != nil {
				return err
			}
		}
	}
	return nil
}

// Prepares the temp and out directories for a target
func prepareDirectories(target *core.BuildTarget) error {
	if err := prepareDirectory(target.TmpDir(), true); err != nil {
		return err
	}
	if err := prepareOutputDirectories(target); err != nil {
		return err
	}
	return prepareDirectory(target.OutDir(), false)
}

func prepareDirectory(directory string, remove bool) error {
	if remove {
		if err := fs.RemoveAll(directory); err != nil {
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
func prepareSources(state *core.BuildState, graph *core.BuildGraph, target *core.BuildTarget) error {
	for src, tmp := range core.IterSources(state, graph, target, false) {
		if err := core.PrepareSource(src, tmp); err != nil {
			return err
		}
	}
	if target.Stamp {
		if err := fs.WriteFile(bytes.NewReader(core.StampFile(state.Config, target)), filepath.Join(target.TmpDir(), target.StampFileName()), 0644); err != nil {
			return err
		}
	}
	if target.SrcListFiles {
		for slf := range target.SourceListFiles(graph) {
			if err := fs.WriteFile(bytes.NewReader(slf.Content), filepath.Join(target.TmpDir(), slf.Dirname, slf.Filename), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// addOutputDirectoriesToBuildOutput moves all the files from the output dirs into the root of the build temp dir
// and adds them as outputs to the build target
func addOutputDirectoriesToBuildOutput(target *core.BuildTarget) ([]string, error) {
	outs := make([]string, 0, len(target.OutputDirectories))
	for _, dir := range target.OutputDirectories {
		o, err := addOutputDirectoryToBuildOutput(target, dir)
		if err != nil {
			return nil, fmt.Errorf("failed to move output dir (%s) contents to rule root: %w", dir, err)
		}
		outs = append(outs, o...)
	}
	return outs, nil
}

func addOutputDirectoryToBuildOutput(target *core.BuildTarget, dir core.OutputDirectory) ([]string, error) {
	fullDir := filepath.Join(target.TmpDir(), dir.Dir())

	files, err := os.ReadDir(fullDir)
	if err != nil {
		return nil, err
	}

	var outs []string
	for _, f := range files {
		from := filepath.Join(fullDir, f.Name())
		to := filepath.Join(target.TmpDir(), f.Name())

		if dir.ShouldAddFiles() {
			newOuts, err := copyOutDir(target, from, to)
			if err != nil {
				return nil, err
			}

			outs = append(outs, newOuts...)
		} else {
			target.AddOutput(f.Name())
			outs = append(outs, f.Name())

			if err := os.Rename(from, to); err != nil {
				return nil, err
			}
		}
	}
	return outs, nil
}

func copyOutDir(target *core.BuildTarget, from string, to string) ([]string, error) {
	relativeToTmpdir := func(path string) string {
		return strings.TrimPrefix(strings.TrimPrefix(path, target.TmpDir()), "/")
	}

	var outs []string

	info, err := os.Lstat(from)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		err := fs.Walk(from, func(name string, isDir bool) error {
			dest := filepath.Join(to, name[len(from):])
			if isDir {
				return os.MkdirAll(dest, fs.DirPermissions)
			}

			outName := relativeToTmpdir(dest)
			outs = append(outs, outName)
			target.AddOutput(outName)

			return os.Rename(name, dest)
		})
		return outs, err
	}
	outs = append(outs, relativeToTmpdir(to))
	target.AddOutput(outs[0])
	return outs, os.Rename(from, to)
}

func moveOutputs(state *core.BuildState, target *core.BuildTarget) ([]string, bool, error) {
	changed := false
	tmpDir := target.TmpDir()
	outDir := target.OutDir()
	outs := target.Outputs()
	allOuts := make([]string, len(outs), len(outs)+len(target.OutputDirectories))
	for i, output := range outs {
		allOuts[i] = output
		tmpOutput := filepath.Join(tmpDir, target.GetTmpOutput(output))
		realOutput := filepath.Join(outDir, output)
		if !core.PathExists(tmpOutput) {
			return nil, true, fmt.Errorf("rule %s failed to create output %s", target.Label, tmpOutput)
		}
		outputChanged, err := moveOutput(state, target, tmpOutput, realOutput)
		if err != nil {
			return nil, true, fmt.Errorf("failed to move output %s: %w", output, err)
		}
		changed = changed || outputChanged
	}
	for ep, out := range target.EntryPoints {
		if !fs.PathExists(filepath.Join(target.OutDir(), out)) {
			return nil, true, fmt.Errorf("failed to produce output %v for entry point %v", out, ep)
		}
	}
	if changed {
		log.Debug("Outputs for %s have changed", target.Label)
	} else {
		log.Debug("Outputs for %s are unchanged", target.Label)
	}
	// Optional outputs get moved but don't contribute to the hash or for incrementality.
	// Glob patterns are supported on these.
	for _, output := range fs.Glob(fs.HostFS, state.Config.Parse.BuildFileName, tmpDir, target.OptionalOutputs, nil, true) {
		log.Debug("Discovered optional output %s", output)
		tmpOutput := filepath.Join(tmpDir, output)
		realOutput := filepath.Join(outDir, output)
		if _, err := moveOutput(state, target, tmpOutput, realOutput); err != nil {
			return nil, changed, err
		}
		allOuts = append(allOuts, output)
	}
	return allOuts, changed, nil
}

func moveOutput(state *core.BuildState, target *core.BuildTarget, tmpOutput, realOutput string) (bool, error) {
	// hash the file
	newHash, err := state.PathHasher.Hash(tmpOutput, false, true, false)
	if err != nil {
		return true, err
	}
	if fs.PathExists(realOutput) {
		if oldHash, err := state.PathHasher.Hash(realOutput, false, true, false); err != nil {
			return true, err
		} else if bytes.Equal(oldHash, newHash) {
			// We already have the same file in the current location. Don't bother moving it.
			log.Debug("Checking %s vs. %s, hashes match", tmpOutput, realOutput)
			return false, nil
		}
		if err := fs.RemoveAll(realOutput); err != nil {
			return true, err
		}
	}
	state.PathHasher.MoveHash(tmpOutput, realOutput)
	// Check if we need a directory for this output.
	dir := filepath.Dir(realOutput)
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
		out := filepath.Join(target.OutDir(), output)
		if err := fs.RemoveAll(out); err != nil {
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
		for dir := filename; dir != "." && dir != "/" && filepath.Base(dir) != "plz-out"; dir = filepath.Dir(dir) {
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
		if state.NeedHashesOnly && state.IsOriginalTargetOrParent(target) {
			log.Info("Updated hash for %v: %v", target, string(hash))
		} else {
			if state.VerifyHashes {
				return nil, err
			}
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
			// if the output is a directory, apply output mode to the file instead of the directory
			err := fs.Walk(output, func(path string, isDir bool) error {
				if isDir {
					return nil
				}
				return os.Chmod(path, target.OutMode())
			})
			if err != nil {
				return nil, fmt.Errorf("failed to mark rule output as binary: %w", err)
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
	if len(outs) == 1 && fs.FileExists(outs[0]) {
		return outputHash(target, outs, h.State.PathHasher, nil)
	}
	return outputHash(target, outs, h.State.PathHasher, h.State.PathHasher.NewHash)
}

// outputHash is a more general form of OutputHash that allows different hashing strategies.
// For example, one could choose to use sha256 instead of our usual sha1.
func outputHash(target *core.BuildTarget, outputs []string, hasher *fs.PathHasher, combine func() hash.Hash) ([]byte, error) {
	if combine == nil {
		// Must be a single output, just hash that directly.
		return hasher.Hash(outputs[0], true, !target.IsFilegroup, target.HashLastModified())
	}
	h := combine()
	for _, filename := range outputs {
		// NB. Always force a recalculation of the output hashes here. Memoisation is not
		//     useful because by definition we are rebuilding a target, and can actively hurt
		//     in cases where we compare the retrieved cache artifacts with what was there before.
		h2, err := hasher.Hash(filename, true, !target.IsFilegroup, target.HashLastModified())
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

// checkRuleHashes verifies the hash of output files for a rule match the ones set on it.
func checkRuleHashes(state *core.BuildState, target *core.BuildTarget, hash []byte) error {
	if len(target.Hashes) == 0 {
		return nil // nothing to check
	}
	outputs := target.FullOutputs()
	hashes := target.UnprefixedHashes()
	// Check if the hash we've already calculated matches any of these before we go off
	// trying any other combinations.
	hashStr := hex.EncodeToString(hash)
	for _, h := range hashes {
		if h == hashStr {
			return nil
		}
	}
	combine := len(outputs) != 1
	validHashes, valid := checkRuleHashesOfType(target, hashes, outputs, state.OutputHashCheckers(), combine)
	if valid {
		return nil
	}
	if len(target.Hashes) == 1 {
		return fmt.Errorf("Bad output hash for rule %s, expected %s, but was: \n\t%s",
			target.Label, target.Hashes[0], strings.Join(validHashes, "\n\t"))
	}
	return fmt.Errorf("Bad output hash for rule %s, expected one of: \n\t%s\nbut was \n\t%s",
		target.Label, strings.Join(target.Hashes, "\n\t"), strings.Join(validHashes, "\n\t"))
}

// checkRuleHashesOfType checks any hashes on this rule of a single type.
// Currently we support SHA-1 and SHA-256, and also have specialisations for the case
// where a target has a single output so as not to double-hash it.
// It is a bit fiddly, but is organised this way to avoid calculating hashes of
// unused types unnecessarily since that could get quite expensive.
func checkRuleHashesOfType(target *core.BuildTarget, hashes, outputs []string, hashers []*fs.PathHasher, combine bool) ([]string, bool) {
	validHashes := make([]string, len(hashers))

	for i, hasher := range hashers {
		var combiner func() hash.Hash
		if combine {
			combiner = hasher.NewHash
		}
		bhash, _ := outputHash(target, outputs, hasher, combiner)
		hashString := hex.EncodeToString(bhash)
		validHashes[i] = fmt.Sprintf("%s: %s", hasher.AlgoName(), hashString)

		for _, h := range hashes {
			if len(h) == hasher.Size()*2 { // Check if the hash is of the right algorithm; 2x because of hex encoding
				if hashString == h {
					return nil, true
				}
			}
		}
	}

	return validHashes, false
}

// Runs the post-build function for a target.
// In some cases it may have already run; if so we compare the previous output and warn
// if the two differ (they must be deterministic to ensure it's a pure function, since there
// are a few different paths through here and we guarantee to only run them once).
func runPostBuildFunction(state *core.BuildState, target *core.BuildTarget, output, prevOutput string) error {
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
	return state.Parser.RunPostBuildFunction(state, target, output)
}

// checkLicences checks the licences for the target match what we've accepted / rejected in the config
// and panics if they don't match.
func checkLicences(state *core.BuildState, target *core.BuildTarget) {
	if _, err := target.CheckLicences(state.Config); err != nil {
		panic(err)
	}
}

// buildLinks builds links from the given target if it's labelled appropriately.
// For example, Go targets may link themselves into plz-out/go/src etc.
func buildLinks(state *core.BuildState, target *core.BuildTarget) {
	buildLinksOfType(state, target, "link:", false, os.Symlink)
	buildLinksOfType(state, target, "hlink:", false, os.Link)

	// Directly link to the path of the label for these (i.e. don't append out to the destination dir)
	buildLinksOfType(state, target, "dlink:", true, os.Symlink)
	buildLinksOfType(state, target, "dhlink:", true, os.Link)

	if state.Config.ShouldLinkGeneratedSources() && target.HasLabel("codegen") {
		generate.LinkGeneratedSources(state, []core.BuildLabel{target.Label})
	}
}

func buildLinksOfType(state *core.BuildState, target *core.BuildTarget, prefix string, direct bool, f fs.LinkFunc) {
	if labels := target.PrefixedLabels(prefix); len(labels) > 0 {
		env := core.TargetEnvironment(state, target)
		for _, dest := range labels {
			destDir := filepath.Join(core.RepoRoot, os.Expand(dest, env.ReplaceEnvironment))
			srcDir := filepath.Join(core.RepoRoot, target.OutDir())
			for _, out := range target.Outputs() {
				if direct {
					fs.LinkDestination(filepath.Join(srcDir, out), destDir, f)
				} else {
					fs.LinkIfNotExists(filepath.Join(srcDir, out), filepath.Join(destDir, out), f)
				}
			}
		}
	}
}

// fetchRemoteFile fetches a remote file from a URL.
// This is a builtin for better efficiency and more control over the whole process.
func fetchRemoteFile(state *core.BuildState, target *core.BuildTarget) error {
	httpClientOnce.Do(func() {
		httpClient = retryablehttp.NewClient()
		httpClient.Logger = &cli.HTTPLogWrapper{Log: log}

		if state.Config.Build.HTTPProxy != "" {
			httpClient.HTTPClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(state.Config.Build.HTTPProxy.AsURL()),
			}
		}

		httpClient.HTTPClient.Timeout = time.Duration(state.Config.Build.Timeout)
		httpClientLimiter = make(chan struct{}, state.Config.Build.ParallelDownloads)
	})

	if err := prepareDirectory(target.OutDir(), false); err != nil {
		return err
	}
	if err := prepareDirectory(target.TmpDir(), false); err != nil {
		return err
	}
	var err error
	for _, src := range target.Sources {
		if e := fetchOneRemoteFile(state, target, src.String()); e != nil {
			err = multierror.Append(err, e)
			audit.WriteRemoteFile(target.Label.Name, src.String(), false, err.Error())
		} else {
			audit.WriteRemoteFile(target.Label.Name, src.String(), true, "")
			return nil
		}
	}
	return err
}

func fetchOneRemoteFile(state *core.BuildState, target *core.BuildTarget, url string) error {
	httpClientLimiter <- struct{}{}
	defer func() { <-httpClientLimiter }()

	env := core.BuildEnvironment(state, target, filepath.Join(core.RepoRoot, target.TmpDir()))
	url = os.Expand(url, env.ReplaceEnvironment)
	tmpPath := filepath.Join(target.TmpDir(), target.Outputs()[0])
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if strings.HasPrefix(url, "file://") {
		filename := strings.TrimPrefix(url, "file://")
		if !filepath.IsAbs(filename) {
			return fmt.Errorf("URL %s must be an absolute path", url)
		} else if strings.HasPrefix(filename, core.RepoRoot) {
			return fmt.Errorf("URL %s is within the repo, you cannot use remote_file for this", url)
		}
		fromfile, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("Error copying %s: %w", url, err)
		}
		defer fromfile.Close()
		if _, err := io.Copy(f, fromfile); err != nil {
			return fmt.Errorf("Error copying %s: %w", url, err)
		}
		return nil
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if err := setHeaders(req, target, env); err != nil {
		return err
	}

	rreq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(rreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		bs, _ := io.ReadAll(resp.Body)
		if len(bs) != 0 {
			log.Debug("Error retrieving %s: %s, Body:\n%s", url, resp.Status, string(bs))
		}
		return fmt.Errorf("Error retrieving %s: %s", url, resp.Status)
	}
	var r io.Reader = resp.Body
	if length := resp.Header.Get("Content-Length"); length != "" {
		if i, err := strconv.Atoi(length); err == nil {
			atomic.StoreUint64(&target.FileSize, uint64(i))
			r = &progressReader{Reader: resp.Body, Target: target, Total: float32(i)}
			target.ShowProgress() // Required for it to actually display
		}
	}
	h := state.PathHasher.NewHash()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		return err
	}
	state.PathHasher.SetHash(tmpPath, h.Sum(nil))
	return nil
}

// setHeaders sets up all the headers we should send on remote_file() requests, including User-Agent and any user
// defined ones.
func setHeaders(req *http.Request, target *core.BuildTarget, env core.BuildEnv) error {
	req.Header.Set("User-Agent", "please.build/"+version.PleaseVersion)

	param := func(str string) (string, string) {
		if !strings.HasPrefix(str, "remote_file:") {
			return "", ""
		}
		str = strings.TrimPrefix(str, "remote_file:")
		i := strings.IndexRune(str, ':')
		return str[:i], str[(i + 1):]
	}

	userName := ""
	password := ""
	for _, l := range target.Labels {
		param, value := param(l)
		switch param {
		case "":
			continue
		case "header":
			k, v := header(value)
			v = os.Expand(v, env.ReplaceEnvironment)
			req.Header.Set(k, v)
		case "secret_header":
			k, v := header(value)
			b, err := os.ReadFile(fs.ExpandHomePath(v))
			if err != nil {
				return fmt.Errorf("failed to read secret file: %v", err)
			}

			req.Header.Set(k, string(b))
		case "username":
			userName = value
		case "password_file":
			p, err := os.ReadFile(fs.ExpandHomePath(value))
			if err != nil {
				return fmt.Errorf("failed to read password file: %v", err)
			}
			password = string(p)
		default:
			return fmt.Errorf("unknown remote file label: %v", l)
		}
	}

	if userName != "" || password != "" {
		req.SetBasicAuth(userName, password)
	}
	return nil
}

func header(str string) (string, string) {
	i := strings.IndexRune(str, ':')
	return str[:i], str[(i + 1):]
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
	r.Target.Progress.Store(100.0 * r.Done / r.Total)
	return n, err
}

// build builds a target locally, it errors if a remote worker is needed since this has beeen removed.
func build(state *core.BuildState, target *core.BuildTarget, inputHash []byte) (*core.BuildMetadata, error) {
	metadata := new(core.BuildMetadata)

	workerCmd, _, localCmd, err := core.WorkerCommandAndArgs(state, target)
	if err != nil {
		return nil, err
	} else if workerCmd == "" {
		metadata.Stdout, err = runBuildCommand(state, target, localCmd, inputHash)
		return metadata, err
	}
	return nil, fmt.Errorf("Persistent workers are no longer supported, found worker command: %s", workerCmd)
}
