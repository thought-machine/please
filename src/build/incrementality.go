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
	"encoding/gob"
	"fmt"
	"hash"
	"os"
	"path/filepath"
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
	// If the metadata file containing the std-out and additional outputs doesn't exist, rebuild
	if !fs.FileExists(targetBuildMetadataFileName(target)) {
		log.Debug("Need to rebuild %s, metadata file is missing", target.Label)
		return true
	}
	oldHashes := readRuleHashFromXattrs(state, target, postBuild)
	if !bytes.Equal(oldHashes.config, state.Hashes.Config) {
		if len(oldHashes.config) == 0 {
			// Small nicety to make it a bit clearer what's going on.
			log.Debug("Need to build %s, outputs aren't there", target.Label)
		} else {
			log.Debug("Need to rebuild %s, config has changed (was %s, need %s)", target.Label, b64(oldHashes.config), b64(state.Hashes.Config))
		}
		return true
	}
	newRuleHash := RuleHash(state, target, false, postBuild)
	if !bytes.Equal(oldHashes.rule, newRuleHash) {
		log.Debug("Need to rebuild %s, rule has changed (was %s, need %s)", target.Label, b64(oldHashes.rule), b64(newRuleHash))
		return true
	}
	newSourceHash, err := sourceHash(state, target)
	if err != nil || !bytes.Equal(oldHashes.source, newSourceHash) {
		log.Debug("Need to rebuild %s, sources have changed (was %s, need %s)", target.Label, b64(oldHashes.source), b64(newSourceHash))
		return true
	}
	newSecretHash, err := secretHash(state, target)
	if err != nil || !bytes.Equal(oldHashes.secret, newSecretHash) {
		log.Debug("Need to rebuild %s, secrets have changed (was %s, need %s)", target.Label, b64(oldHashes.secret), b64(newSecretHash))
		return true
	}

	// Check the outputs of this rule exist. This would only happen if the user had
	// removed them but it's incredibly aggravating if you remove an output and the
	// rule won't rebuild itself.
	for _, output := range target.Outputs() {
		realOutput := filepath.Join(target.OutDir(), output)
		if !core.PathExists(realOutput) {
			log.Debug("Output %s doesn't exist for rule %s; will rebuild.", realOutput, target.Label)
			return true
		}
	}
	// Maybe we've forced a rebuild. Do this last; might be interesting to see if it needed building anyway.
	return state.ShouldRebuild(target)
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
	for source := range core.IterSources(state, state.Graph, target, false) {
		result, err := state.PathHasher.Hash(source.Src, false, true, false)
		if err != nil {
			return nil, err
		}
		h.Write(result)
		h.Write([]byte(source.Src))
	}
	for _, tool := range target.AllTools() {
		for _, path := range tool.FullPaths(state.Graph) {
			result, err := state.PathHasher.Hash(path, false, true, false)
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
	if runtime || (postBuild && target.BuildCouldModifyTarget()) {
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
	hashOptionalBool(h, target.IsSubrepo)
	hashOptionalBool(h, target.Sandbox)

	// Note that we only hash the current command here; whatever's set in commands that we're not going
	// to run is uninteresting to us.
	h.Write([]byte(target.GetCommand(state)))

	hashBool(h, target.NeedsTransitiveDependencies)
	hashBool(h, target.OutputIsComplete)
	hashBool(h, target.Stamp)
	hashBool(h, target.IsFilegroup)
	hashBool(h, target.IsTextFile)
	hashBool(h, target.IsRemoteFile)
	hashBool(h, target.Local)
	hashOptionalBool(h, target.ExitOnError)
	for _, require := range target.Requires {
		h.Write([]byte(require))
	}
	// Indeterminate iteration order, yay...
	var provides []string
	for k := range target.Provides {
		provides = append(provides, k)
	}
	sort.Strings(provides)
	for _, p := range provides {
		h.Write([]byte(p))
		for _, l := range target.Provides[p] {
			h.Write([]byte(l.String()))
		}
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

	for _, o := range target.OutputDirectories {
		h.Write([]byte(o))
	}

	hashMap(h, target.EntryPoints)
	hashMap(h, target.Env)

	h.Write([]byte(target.FileContent))

	// Hash the test and runtime fields
	if runtime {
		for _, datum := range target.AllData() {
			h.Write([]byte(datum.String()))
		}
		if target.IsTest() {
			for _, output := range target.Test.Outputs {
				h.Write([]byte(output))
			}
			hashOptionalBool(h, target.Test.Sandbox)
			h.Write([]byte(target.GetTestCommand(state)))
		}
	}

	return h.Sum(nil)
}

func hashMap(writer hash.Hash, eps map[string]string) {
	keys := make([]string, 0, len(eps))
	for ep := range eps {
		keys = append(keys, ep)
	}
	sort.Strings(keys)
	for _, ep := range keys {
		writer.Write([]byte(ep + "=" + eps[ep]))
	}
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

type ruleHashes struct {
	rule, config, source, secret []byte
	postBuildHash                bool
}

// readRuleHashFromXattrs reads the hash of a file using xattrs.
// If postBuild is true then the rule hash will be the post-build one if present.
func readRuleHashFromXattrs(state *core.BuildState, target *core.BuildTarget, postBuild bool) ruleHashes {
	var h []byte
	for _, output := range target.FullOutputs() {
		b := fs.ReadAttr(output, xattrName, state.XattrsSupported)
		if b == nil {
			return ruleHashes{}
		} else if h != nil && !bytes.Equal(h, b) {
			// Not an error; we could warn but it's possible to get here legitimately so
			// just return nothing.
			return ruleHashes{}
		}
		h = b
	}
	if h == nil {
		// If the target could be modified during build, we might have written the hash on the build MD file.
		// Only works for pre-build, though.
		if target.BuildCouldModifyTarget() && !postBuild {
			h = fs.ReadAttr(targetBuildMetadataFileName(target), xattrName, state.XattrsSupported)
			if h == nil {
				return ruleHashes{}
			}
		} else {
			// Try the fallback file; target might not have had any outputs, for example.
			h = fs.ReadAttrFile(filepath.Join(target.OutDir(), target.Label.Name))
			if h == nil {
				return ruleHashes{}
			}
		}
	}
	if postBuild {
		return ruleHashes{
			rule:          h[hashLength : 2*hashLength],
			config:        h[2*hashLength : 3*hashLength],
			source:        h[3*hashLength : 4*hashLength],
			secret:        h[4*hashLength : fullHashLength],
			postBuildHash: true,
		}
	}
	return ruleHashes{
		rule:   h[0:hashLength],
		config: h[2*hashLength : 3*hashLength],
		source: h[3*hashLength : 4*hashLength],
		secret: h[4*hashLength : fullHashLength],
	}
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
		return fs.RecordAttrFile(filepath.Join(target.OutDir(), target.Label.Name), hash)
	}
	for _, output := range outputs {
		if err := fs.RecordAttr(output, hash, xattrName, state.XattrsSupported); err != nil {
			return err
		}
	}
	if fs.FileExists(targetBuildMetadataFileName(target)) {
		return fs.RecordAttr(targetBuildMetadataFileName(target), hash, xattrName, state.XattrsSupported)
	}
	return nil
}

func targetBuildMetadataFileName(target *core.BuildTarget) string {
	return filepath.Join(target.OutDir(), target.TargetBuildMetadataFileName())
}

// loadTargetMetadata retrieves the target metadata from a file in the output directory of this target
func loadTargetMetadata(target *core.BuildTarget) (*core.BuildMetadata, error) {
	file, err := os.Open(targetBuildMetadataFileName(target))
	if err != nil {
		return nil, err
	}

	defer file.Close()

	md := new(core.BuildMetadata)

	reader := gob.NewDecoder(file)
	if err := reader.Decode(&md); err != nil {
		return nil, err
	}

	return md, nil
}

// StoreTargetMetadata stores the target metadata into a file in the output directory of the target.
func StoreTargetMetadata(target *core.BuildTarget, md *core.BuildMetadata) error {
	filename := targetBuildMetadataFileName(target)
	if err := os.RemoveAll(filename); err != nil {
		return fmt.Errorf("failed to remove existing %s build metadata file: %w", target.Label, err)
	} else if err := os.MkdirAll(filepath.Dir(filename), core.DirPermissions); err != nil {
		return fmt.Errorf("Failed to create directory for build metadata file for %s: %w", target, err)
	}

	mdFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create new %s build metadata file: %w", target.Label, err)
	}

	defer mdFile.Close()

	writer := gob.NewEncoder(mdFile)
	if err := writer.Encode(md); err != nil {
		return fmt.Errorf("failed to encode %s build metadata file: %w", target.Label, err)
	}
	return nil
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
func RuntimeHash(state *core.BuildState, target *core.BuildTarget, testRun int) ([]byte, error) {
	hash := append(RuleHash(state, target, true, false), RuleHash(state, target, true, true)...)
	hash = append(hash, state.Hashes.Config...)
	h := sha1.New()
	for source := range core.IterRuntimeFiles(state.Graph, target, true, target.TestDir(testRun)) {
		result, err := state.PathHasher.Hash(source.Src, false, true, false)
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
	if state.RemoteClient != nil && !target.Local {
		state.RemoteClient.PrintHashes(target, false)
		return
	}
	fmt.Printf("%s:\n", target.Label)
	fmt.Printf("  Config: %s\n", b64(state.Hashes.Config))
	fmt.Printf("    Rule: %s (pre-build)\n", b64(RuleHash(state, target, false, false)))
	fmt.Printf("    Rule: %s (post-build)\n", b64(RuleHash(state, target, false, true)))
	fmt.Printf("  Source: %s\n", b64(mustSourceHash(state, target)))
	// Note that the logic here mimics sourceHash, but I don't want to pollute that with
	// optional printing nonsense since it's on our hot path.
	for source := range core.IterSources(state, state.Graph, target, false) {
		fmt.Printf("  Source: %s: %s\n", source.Src, b64(state.PathHasher.MustHash(source.Src, target.HashLastModified())))
	}
	for _, tool := range target.AllTools() {
		if label, ok := tool.Label(); ok {
			fmt.Printf("    Tool: %s: %s\n", label, b64(mustShortTargetHash(state, state.Graph.TargetOrDie(label))))
		} else {
			fmt.Printf("    Tool: %s: %s\n", tool, b64(state.PathHasher.MustHash(tool.FullPaths(state.Graph)[0], target.HashLastModified())))
		}
	}
}

// secretHash calculates a hash for any secrets of a target.
func secretHash(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	if len(target.Secrets) == 0 {
		return noSecrets, nil
	}
	h := sha1.New()
	for _, secret := range target.Secrets {
		ph, err := state.PathHasher.Hash(secret, false, false, false)
		if err != nil && os.IsNotExist(err) {
			return noSecrets, nil // Not having the secrets is not an error yet.
		} else if err != nil {
			return nil, err
		}
		h.Write(ph)
	}
	return h.Sum(nil), nil
}
