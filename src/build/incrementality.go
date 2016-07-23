// Utilities to help with incremental builds.
//
// There are four things we consider for each rule:
//  - the global config, any change to which invalidates artifacts
//    (it's too hard to work out which bits affect which rules)
//  - the rule definition itself (the command to run, etc)
//  - any input files it might have
//  - any dependencies.
//
// If all of those are the same as the last time the rule was run,
// we can safely assume that the output will be the same this time
// and so we don't have to re-run it again.

package build

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"core"
)

const hashLength = sha1.Size

// Length of the hash file we write
const hashFileLength = 4 * hashLength

// Used to write something when we need to indicate a boolean in a hash. Can be essentially
// any value as long as they're different from one another.
var boolTrueHashValue = []byte{2}
var boolFalseHashValue = []byte{1}

// Return true if the rule needs building, false if the existing outputs are OK.
func needsBuilding(state *core.BuildState, target *core.BuildTarget, postBuild bool) bool {
	// Check the dependencies first, because they don't need any disk I/O.
	if target.NeedsTransitiveDependencies {
		if anyDependencyHasChanged(target) {
			return true // one of the transitive deps has changed, need to rebuild
		}
	} else {
		for _, dep := range target.Dependencies() {
			if dep.State() < core.Unchanged {
				log.Debug("Need to rebuild %s, %s has changed", target.Label, dep.Label)
				return true // dependency has just been rebuilt, do this too.
			}
		}
	}
	oldRuleHash, oldConfigHash, oldSourceHash := readRuleHashFile(ruleHashFileName(target), postBuild)
	if !bytes.Equal(oldConfigHash, state.Hashes.Config) {
		if len(oldConfigHash) == 0 {
			// Small nicety to make it a bit clearer what's going on.
			log.Debug("Need to build %s, outputs aren't there", target.Label)
		} else {
			log.Debug("Need to rebuild %s, config has changed (was %s, need %s)", target.Label, b64(oldConfigHash), b64(state.Hashes.Config))
		}
		return true
	}
	newRuleHash := RuleHash(target, false, postBuild)
	if !bytes.Equal(oldRuleHash, newRuleHash) {
		log.Debug("Need to rebuild %s, rule has changed (was %s, need %s)", target.Label, b64(oldRuleHash), b64(newRuleHash))
		return true
	}
	newSourceHash, err := sourceHash(state.Graph, target)
	if err != nil || !bytes.Equal(oldSourceHash, newSourceHash) {
		log.Debug("Need to rebuild %s, sources have changed (was %s, need %s)", target.Label, b64(oldSourceHash), b64(newSourceHash))
		return true
	}
	// Check the outputs of this rule exist. This would only happen if the user had
	// removed them but it's incredibly aggravating if you remove an output and the
	// rule won't rebuild itself.
	for _, output := range target.Outputs() {
		realOutput := path.Join(target.OutDir(), output)
		if !core.PathExists(realOutput) {
			log.Debug("Output %s doesn't exist for rule %s; will rebuild.", realOutput, target.Label)
			return true
		}
	}
	// Maybe we've forced a rebuild. Do this last; might be interesting to see if it needed building anyway.
	return state.ForceRebuild && state.IsOriginalTarget(target.Label)
}

// b64 base64 encodes a string of bytes for printing.
func b64(b []byte) string {
	if len(b) == 0 {
		return "<not found>"
	}
	return base64.RawStdEncoding.EncodeToString(b)
}

// Returns true if any transitive dependency of this target has changed.
func anyDependencyHasChanged(target *core.BuildTarget) bool {
	done := map[core.BuildLabel]bool{}
	var inner func(*core.BuildTarget) bool
	inner = func(dependency *core.BuildTarget) bool {
		done[dependency.Label] = true
		if dependency != target && dependency.State() < core.Unchanged {
			return true
		} else if !dependency.OutputIsComplete || dependency == target {
			for _, dep := range dependency.Dependencies() {
				if !done[dep.Label] {
					if inner(dep) {
						log.Debug("Need to rebuild %s, %s has changed", target.Label, dep.Label)
						return true
					}
				}
			}
		}
		return false
	}
	return inner(target)
}

// Calculate the hash of all sources of this rule
func sourceHash(graph *core.BuildGraph, target *core.BuildTarget) ([]byte, error) {
	h := sha1.New()
	for source := range core.IterSources(graph, target) {
		result, err := pathHash(source.Src, false)
		if err != nil {
			return result, err
		}
		h.Write(result)
	}
	// Note that really it would be more correct to hash the outputs of these rules
	// in the same way we calculate a hash of sources for the rule, but that is
	// impractical for some cases (notably npm) where tools can be very large.
	// Instead we assume calculating the target hash is sufficient.
	for _, tool := range target.Tools {
		h.Write(mustTargetHash(core.State, graph.TargetOrDie(tool)))
	}
	return h.Sum(nil), nil
}

// Used to memoize the results of pathHash so we don't hash the same files multiple times.
var pathHashMemoizer = map[string][]byte{}
var pathHashMutex sync.RWMutex // Of course it will be accessed concurrently.

// Calculate the hash of a single path which might be a file or a directory
// This is the memoized form that only hashes each path once, unless recalc is true in which
// case it will force a recalculation of the hash.
func pathHash(path string, recalc bool) ([]byte, error) {
	path = ensureRelative(path)
	if !recalc {
		pathHashMutex.RLock()
		cached, present := pathHashMemoizer[path]
		pathHashMutex.RUnlock()
		if present {
			return cached, nil
		}
	}
	result, err := pathHashImpl(path)
	if err == nil {
		pathHashMutex.Lock()
		pathHashMemoizer[path] = result
		pathHashMutex.Unlock()
	}
	return result, err
}

func pathHashImpl(path string) ([]byte, error) {
	h := sha1.New()
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		// Dereference symlink and try again
		deref, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		// Write something indicating this was a link; important so we rebuild correctly
		// when eg. a filegroup is changed from link=False to link=True.
		// Don't want to hash all file mode bits, the others could change depending on
		// whether we retrieved from cache or not so they're probably a bit too fragile.
		h.Write(boolTrueHashValue)
		d, err := pathHashImpl(deref)
		h.Write(d)
		return h.Sum(nil), err
	}
	if err == nil && info.IsDir() {
		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			} else if !info.IsDir() {
				return fileHash(&h, path)
			} else {
				return nil
			}
		})
	} else {
		err = fileHash(&h, path) // let this handle any other errors
	}
	return h.Sum(nil), err
}

// movePathHash is used when we move files from tmp to out and there was one there before; that's
// the only case in which the hash of a filepath could change.
func movePathHash(oldPath, newPath string, copy bool) {
	oldPath = ensureRelative(oldPath)
	newPath = ensureRelative(newPath)
	pathHashMutex.Lock()
	pathHashMemoizer[newPath] = pathHashMemoizer[oldPath]
	// If the path is in plz-out/tmp we aren't ever going to use it again, so free some space.
	if !copy && strings.HasPrefix(oldPath, core.TmpDir) {
		delete(pathHashMemoizer, oldPath)
	}
	pathHashMutex.Unlock()
}

// ensureRelative ensures a path is relative to the repo root.
// This is important for getting best performance from memoizing the path hashes.
func ensureRelative(path string) string {
	if strings.HasPrefix(path, core.RepoRoot) {
		return strings.TrimLeft(strings.TrimPrefix(path, core.RepoRoot), "/")
	}
	return path
}

// Calculate the hash of a single file
func fileHash(h *hash.Hash, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(*h, file)
	file.Close()
	return err
}

// RuleHash calculates a hash for the relevant bits of this rule that affect its output.
// Optionally it can include parts of the rule that affect runtime (most obviously test-time).
// Note that we have to hash on the declared fields, we obviously can't hash pointers etc.
// incrementality_test will warn if new fields are added to the struct but not here.
func RuleHash(target *core.BuildTarget, runtime, postBuild bool) []byte {
	if runtime || (postBuild && target.PostBuildFunction != 0) {
		return ruleHash(target, runtime)
	}
	// Non-post-build hashes get stored on the target itself.
	if len(target.RuleHash) != 0 {
		return target.RuleHash
	}
	target.RuleHash = ruleHash(target, false) // This is never a runtime hash.
	return target.RuleHash
}

func ruleHash(target *core.BuildTarget, runtime bool) []byte {
	h := sha1.New()
	h.Write([]byte(target.Label.String()))
	for _, dep := range target.DeclaredDependencies() {
		h.Write([]byte(dep.String()))
	}
	for _, vis := range target.Visibility {
		h.Write([]byte(vis.String())) // Doesn't strictly affect the output, but best to be safe.
	}
	for _, hsh := range target.Hashes {
		h.Write([]byte(hsh))
	}
	for _, source := range target.AllSources() {
		h.Write([]byte(source.String()))
	}
	for _, out := range target.DeclaredOutputs() {
		h.Write([]byte(out))
	}
	for _, licence := range target.Licences {
		h.Write([]byte(licence))
	}
	for _, output := range target.TestOutputs {
		h.Write([]byte(output))
	}
	for _, label := range target.Labels {
		h.Write([]byte(label))
	}
	hashBool(h, target.IsBinary)
	hashBool(h, target.IsTest)

	// Note that we only hash the current command here; whatever's set in commands that we're not going
	// to run is uninteresting to us.
	h.Write([]byte(target.GetCommand()))

	if runtime {
		h.Write([]byte(target.TestCommand))
		for _, datum := range target.Data {
			h.Write([]byte(datum.String()))
		}
		hashBool(h, target.Containerise)
		if target.ContainerSettings != nil {
			e := gob.NewEncoder(h)
			if err := e.Encode(target.ContainerSettings); err != nil {
				panic(err)
			}
		}
		if target.Containerise {
			h.Write(core.State.Hashes.Containerisation)
		}
	}

	hashBool(h, target.NeedsTransitiveDependencies)
	hashBool(h, target.OutputIsComplete)
	// Should really not be conditional here, but we don't want adding the new flag to
	// change the hash of every single other target everywhere.
	// Might consider removing this the next time we peturb the hashing strategy.
	if target.Stamp {
		hashBool(h, target.Stamp)
	}
	for _, require := range target.Requires {
		h.Write([]byte(require))
	}
	// Indeterminate iteration order, yay...
	languages := []string{}
	for k := range target.Provides {
		languages = append(languages, k)
	}
	sort.Strings(languages)
	for _, lang := range languages {
		h.Write([]byte(lang))
		h.Write([]byte(target.Provides[lang].String()))
	}
	// Obviously we don't include the code pointer because it's a pointer.
	h.Write(target.PreBuildHash)
	h.Write(target.PostBuildHash)
	// The Go version affects the hash for Go targets only.
	if target.HasLabel("go") {
		h.Write([]byte(core.State.Config.Go.GoVersion))
	}
	return h.Sum(nil)
}

func hashBool(writer hash.Hash, b bool) {
	if b {
		writer.Write(boolTrueHashValue)
	} else {
		writer.Write(boolFalseHashValue)
	}
}

// readRuleHashFile reads the contents of a rule hash file into separate byte arrays
// Arrays will be empty if there's an error reading the file.
// If postBuild is true then the rule hash will be the post-build one if present.
func readRuleHashFile(filename string, postBuild bool) ([]byte, []byte, []byte) {
	contents := make([]byte, hashFileLength, hashFileLength)
	file, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warning("Failed to read rule hash file %s: %s", filename, err)
		}
		return nil, nil, nil
	}
	defer file.Close()
	if n, err := file.Read(contents); err != nil {
		log.Warning("Error reading rule hash file %s: %s", filename, err)
		return nil, nil, nil
	} else if n != hashFileLength {
		log.Warning("Unexpected rule hash file length: expected %d bytes, was %d", hashFileLength, n)
		return nil, nil, nil
	}
	if postBuild {
		return contents[hashLength : 2*hashLength], contents[2*hashLength : 3*hashLength], contents[3*hashLength : hashFileLength]
	}
	return contents[0:hashLength], contents[2*hashLength : 3*hashLength], contents[3*hashLength : hashFileLength]
}

// Writes the contents of the rule hash file
func writeRuleHashFile(state *core.BuildState, target *core.BuildTarget) error {
	hash, err := targetHash(state, target)
	if err != nil {
		return err
	}
	file, err := os.Create(ruleHashFileName(target))
	if err != nil {
		return err
	}
	defer file.Close()
	n, err := file.Write(hash)
	if err != nil {
		return err
	} else if n != hashFileLength {
		return fmt.Errorf("Wrote %d bytes to rule hash file; should be %d", n, hashFileLength)
	}
	return nil
}

// Returns the filename we'll store the hashes for this file in.
func ruleHashFileName(target *core.BuildTarget) string {
	return path.Join(target.OutDir(), ".rule_hash_"+target.Label.Name)
}

func postBuildOutputFileName(target *core.BuildTarget) string {
	return path.Join(target.OutDir(), core.PostBuildOutputFileName(target))
}

// For targets that have post-build functions, we have to store and retrieve the target's
// output to feed to it
func loadPostBuildOutput(state *core.BuildState, target *core.BuildTarget) string {
	// Normally filegroups don't have post-build functions, but we use this sometimes for testing.
	if target.IsFilegroup() {
		return ""
	}
	out, err := ioutil.ReadFile(postBuildOutputFileName(target))
	if err != nil {
		panic(err)
	}
	return string(out)
}

func storePostBuildOutput(state *core.BuildState, target *core.BuildTarget, out []byte) {
	filename := postBuildOutputFileName(target)
	if err := os.RemoveAll(filename); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, out, 0644); err != nil {
		panic(err)
	}
}

// targetHash returns the hash for a target and any error encountered while calculating it.
func targetHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	hash := append(RuleHash(target, false, false), RuleHash(target, false, true)...)
	hash = append(hash, state.Hashes.Config...)
	hash2, err := sourceHash(state.Graph, target)
	if err != nil {
		return nil, err
	}
	return append(hash, hash2...), nil
}

// mustTargetHash returns the hash for a target and panics if it can't be calculated.
func mustTargetHash(state *core.BuildState, target *core.BuildTarget) []byte {
	hash, err := targetHash(state, target)
	if err != nil {
		panic(err)
	}
	return hash
}

// mustShortTargetHash returns the hash for a target, shortened to 1/4 length.
func mustShortTargetHash(state *core.BuildState, target *core.BuildTarget) []byte {
	return core.CollapseHash(mustTargetHash(state, target))
}

// RuntimeHash returns the target hash, source hash, config hash & runtime file hash,
// all rolled into one. Essentially this is one hash needed to determine if the runtime
// state is consistent.
func RuntimeHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	hash := append(RuleHash(target, true, false), RuleHash(target, true, true)...)
	hash = append(hash, state.Hashes.Config...)
	sh, err := sourceHash(state.Graph, target)
	if err != nil {
		return nil, err
	}
	h := sha1.New()
	h.Write(sh)
	for source := range core.IterRuntimeFiles(state.Graph, target, true) {
		result, err := pathHash(source.Src, false)
		if err != nil {
			return result, err
		}
		h.Write(result)
	}
	return append(hash, h.Sum(nil)...), nil
}
