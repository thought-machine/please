// Utilities to help with incremental builds.
//
// There are four things we consider for each rule:
//  - the global config, some parts of which affect all rules
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
	"fmt"
	"hash"
	"io/ioutil"
	"os"
	"path"
	"sort"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

const hashLength = sha1.Size

// Tag that we attach for xattrs to store hashes against files.
// Note that we are required to provide the user namespace; that seems to be set implicitly
// by the attr utility, but that is not done for us here.
const xattrName = "user.plz_build"

// Length of the full hash we write, which has multiple parts.
const fullHashLength = 5 * hashLength

// noSecrets is the thing we write when a rule doesn't have any secrets defined.
var noSecrets = []byte{45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45, 45}

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
	oldRuleHash, oldConfigHash, oldSourceHash, oldSecretHash := readRuleHash(state, target, postBuild)
	if !bytes.Equal(oldConfigHash, state.Hashes.Config) {
		if len(oldConfigHash) == 0 {
			// Small nicety to make it a bit clearer what's going on.
			log.Debug("Need to build %s, outputs aren't there", target.Label)
		} else {
			log.Debug("Need to rebuild %s, config has changed (was %s, need %s)", target.Label, b64(oldConfigHash), b64(state.Hashes.Config))
		}
		return true
	}
	newRuleHash := RuleHash(state, target, false, postBuild)
	if !bytes.Equal(oldRuleHash, newRuleHash) {
		log.Debug("Need to rebuild %s, rule has changed (was %s, need %s)", target.Label, b64(oldRuleHash), b64(newRuleHash))
		return true
	}
	newSourceHash, err := sourceHash(state, target)
	if err != nil || !bytes.Equal(oldSourceHash, newSourceHash) {
		log.Debug("Need to rebuild %s, sources have changed (was %s, need %s)", target.Label, b64(oldSourceHash), b64(newSourceHash))
		return true
	}
	newSecretHash, err := secretHash(state, target)
	if err != nil || !bytes.Equal(oldSecretHash, newSecretHash) {
		log.Debug("Need to rebuild %s, secrets have changed (was %s, need %s)", target.Label, b64(oldSecretHash), b64(newSecretHash))
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
	return state.ForceRebuild && (state.IsOriginalTarget(target.Label) || state.IsOriginalTarget(target.Label.Parent()))
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

func mustSourceHash(state *core.BuildState, target *core.BuildTarget) []byte {
	b, err := sourceHash(state, target)
	if err != nil {
		log.Fatalf("%s", err)
	}
	return b
}

// Calculate the hash of all sources of this rule
func sourceHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	h := sha1.New()
	for source := range core.IterSources(state.Graph, target, false) {
		result, err := state.PathHasher.Hash(source.Src, false, true)
		if err != nil {
			return nil, err
		}
		h.Write(result)
		h.Write([]byte(source.Src))
	}
	for _, tool := range target.AllTools() {
		for _, path := range tool.FullPaths(state.Graph) {
			result, err := state.PathHasher.Hash(path, false, true)
			if err != nil {
				return nil, err
			}
			h.Write(result)
		}
	}
	return h.Sum(nil), nil
}

// RuleHash calculates a hash for the relevant bits of this rule that affect its output.
// Optionally it can include parts of the rule that affect runtime (most obviously test-time).
// Note that we have to hash on the declared fields, we obviously can't hash pointers etc.
// incrementality_test will warn if new fields are added to the struct but not here.
func RuleHash(state *core.BuildState, target *core.BuildTarget, runtime, postBuild bool) []byte {
	if runtime || (postBuild && target.PostBuildFunction != nil) {
		return ruleHash(state, target, runtime)
	}
	// Non-post-build hashes get stored on the target itself.
	if len(target.RuleHash) != 0 {
		return target.RuleHash
	}
	target.RuleHash = ruleHash(state, target, false) // This is never a runtime hash.
	return target.RuleHash
}

func ruleHash(state *core.BuildState, target *core.BuildTarget, runtime bool) []byte {
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
	outs := target.DeclaredNamedOutputs()
	for _, name := range target.DeclaredOutputNames() {
		h.Write([]byte(name))
		for _, out := range outs[name] {
			h.Write([]byte(out))
		}
	}
	for _, licence := range target.Licences {
		h.Write([]byte(licence))
	}
	for _, output := range target.TestOutputs {
		h.Write([]byte(output))
	}
	for _, output := range target.OptionalOutputs {
		h.Write([]byte(output))
	}
	for _, label := range target.Labels {
		h.Write([]byte(label))
	}
	for _, secret := range target.Secrets {
		h.Write([]byte(secret))
	}
	hashBool(h, target.IsBinary)
	hashBool(h, target.IsTest)
	hashOptionalBool(h, target.Sandbox)

	// Note that we only hash the current command here; whatever's set in commands that we're not going
	// to run is uninteresting to us.
	h.Write([]byte(target.GetCommand(state)))

	if runtime {
		// Similarly, we only hash the current command here again.
		h.Write([]byte(target.GetTestCommand(state)))
		for _, datum := range target.AllData() {
			h.Write([]byte(datum.String()))
		}
		hashOptionalBool(h, target.TestSandbox)
	}

	hashBool(h, target.NeedsTransitiveDependencies)
	hashBool(h, target.OutputIsComplete)
	// Should really not be conditional here, but we don't want adding the new flag to
	// change the hash of every single other target everywhere.
	// Might consider removing this the next time we peturb the hashing strategy.
	hashOptionalBool(h, target.Stamp)
	hashOptionalBool(h, target.IsFilegroup)
	hashOptionalBool(h, target.IsHashFilegroup)
	hashOptionalBool(h, target.IsRemoteFile)
	hashOptionalBool(h, target.Local)
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
	// We don't need to hash the functions themselves because they get rerun every time -
	// we just need to check whether one is added or removed, which is good since it's
	// nigh impossible to really verify whether it's changed or not (since it may call
	// any amount of other stuff).
	hashBool(h, target.PreBuildFunction != nil)
	hashBool(h, target.PostBuildFunction != nil)
	if target.PassEnv != nil {
		for _, env := range *target.PassEnv {
			h.Write([]byte(env))
			h.Write([]byte{'='})
			h.Write([]byte(os.Getenv(env)))
		}
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

func hashOptionalBool(writer hash.Hash, b bool) {
	if b {
		hashBool(writer, b)
	}
}

// readRuleHash reads the hash of a file using xattrs.
// If postBuild is true then the rule hash will be the post-build one if present.
func readRuleHash(state *core.BuildState, target *core.BuildTarget, postBuild bool) ([]byte, []byte, []byte, []byte) {
	var h []byte
	for _, output := range target.FullOutputs() {
		b := fs.ReadAttr(output, xattrName, state.XattrsSupported)
		if b == nil {
			return nil, nil, nil, nil
		} else if h != nil && !bytes.Equal(h, b) {
			// Not an error; we could warn but it's possible to get here legitimately so
			// just return nothing.
			return nil, nil, nil, nil
		}
		h = b
	}
	if h == nil {
		// If the target has a post-build function, we might have written it there.
		// Only works for pre-build, though.
		if target.PostBuildFunction != nil && !postBuild {
			h = fs.ReadAttr(postBuildOutputFileName(target), xattrName, state.XattrsSupported)
			if h == nil {
				return nil, nil, nil, nil
			}
		} else {
			// Try the fallback file; target might not have had any outputs, for example.
			h = fs.ReadAttrFile(path.Join(target.OutDir(), target.Label.Name))
			if h == nil {
				return nil, nil, nil, nil
			}
		}
	}
	if postBuild {
		return h[hashLength : 2*hashLength], h[2*hashLength : 3*hashLength], h[3*hashLength : 4*hashLength], h[4*hashLength : fullHashLength]
	}
	return h[0:hashLength], h[2*hashLength : 3*hashLength], h[3*hashLength : 4*hashLength], h[4*hashLength : fullHashLength]
}

// writeRuleHash attaches the rule hash to the file to its outputs using xattrs.
func writeRuleHash(state *core.BuildState, target *core.BuildTarget) error {
	hash, err := targetHash(state, target)
	if err != nil {
		return err
	}
	secretHash, err := secretHash(state, target)
	if err != nil {
		return err
	}
	hash = append(hash, secretHash...)
	outputs := target.FullOutputs()
	if len(outputs) == 0 {
		// Target has no outputs, have to use the fallback file.
		return fs.RecordAttrFile(path.Join(target.OutDir(), target.Label.Name), hash)
	}
	for _, output := range outputs {
		if err := fs.RecordAttr(output, hash, xattrName, state.XattrsSupported); err != nil {
			return err
		}
	}
	if target.PostBuildFunction != nil {
		return fs.RecordAttr(postBuildOutputFileName(target), hash, xattrName, state.XattrsSupported)
	}
	return nil
}

// fallbackRuleHashFile returns the filename we'll store the hashes for this file on if we have
// no alternative (for example, if it doesn't have any outputs we have to put them *somewhere*)
func fallbackRuleHashFileName(target *core.BuildTarget) string {
	return path.Join(target.OutDir(), target.Label.Name)
}

func postBuildOutputFileName(target *core.BuildTarget) string {
	return path.Join(target.OutDir(), target.PostBuildOutputFileName())
}

// For targets that have post-build functions, we have to store and retrieve the target's
// output to feed to it
func loadPostBuildOutput(target *core.BuildTarget) (string, error) {
	// Normally filegroups don't have post-build functions, but we use this sometimes for testing.
	if target.IsFilegroup {
		return "", nil
	}
	out, err := ioutil.ReadFile(postBuildOutputFileName(target))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func storePostBuildOutput(target *core.BuildTarget, out []byte) {
	filename := postBuildOutputFileName(target)
	if err := os.RemoveAll(filename); err != nil {
		panic(err)
	} else if err := ioutil.WriteFile(filename, out, 0644); err != nil {
		panic(err)
	}
}

// targetHash returns the hash for a target and any error encountered while calculating it.
func targetHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	hash := append(RuleHash(state, target, false, false), RuleHash(state, target, false, true)...)
	hash = append(hash, state.Hashes.Config...)
	hash2, err := sourceHash(state, target)
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

// RuntimeHash returns the target hash, config hash & runtime file hash,
// all rolled into one. Essentially this is one hash needed to determine if the runtime
// state is consistent.
func RuntimeHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	hash := append(RuleHash(state, target, true, false), RuleHash(state, target, true, true)...)
	hash = append(hash, state.Hashes.Config...)
	h := sha1.New()
	for source := range core.IterRuntimeFiles(state.Graph, target, true) {
		result, err := state.PathHasher.Hash(source.Src, false, true)
		if err != nil {
			return result, err
		}
		h.Write(result)
	}
	return append(hash, h.Sum(nil)...), nil
}

// PrintHashes prints the various hashes for a target to stdout.
// It's used by plz hash --detailed to show a breakdown of the input hashes of a target.
func PrintHashes(state *core.BuildState, target *core.BuildTarget) {
	fmt.Printf("%s:\n", target.Label)
	fmt.Printf("  Config: %s\n", b64(state.Hashes.Config))
	fmt.Printf("    Rule: %s (pre-build)\n", b64(RuleHash(state, target, false, false)))
	fmt.Printf("    Rule: %s (post-build)\n", b64(RuleHash(state, target, false, true)))
	fmt.Printf("  Source: %s\n", b64(mustSourceHash(state, target)))
	// Note that the logic here mimics sourceHash, but I don't want to pollute that with
	// optional printing nonsense since it's on our hot path.
	for source := range core.IterSources(state.Graph, target, false) {
		fmt.Printf("  Source: %s: %s\n", source.Src, b64(state.PathHasher.MustHash(source.Src)))
	}
	for _, tool := range target.AllTools() {
		if label := tool.Label(); label != nil {
			fmt.Printf("    Tool: %s: %s\n", *label, b64(mustShortTargetHash(state, state.Graph.TargetOrDie(*label))))
		} else {
			fmt.Printf("    Tool: %s: %s\n", tool, b64(state.PathHasher.MustHash(tool.FullPaths(state.Graph)[0])))
		}
	}
	if state.RemoteClient != nil {
		state.RemoteClient.PrintHashes(target, false)
	}
}

// secretHash calculates a hash for any secrets of a target.
func secretHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	if len(target.Secrets) == 0 {
		return noSecrets, nil
	}
	h := sha1.New()
	for _, secret := range target.Secrets {
		ph, err := state.PathHasher.Hash(secret, false, false)
		if err != nil && os.IsNotExist(err) {
			return noSecrets, nil // Not having the secrets is not an error yet.
		} else if err != nil {
			return nil, err
		}
		h.Write(ph)
	}
	return h.Sum(nil), nil
}
