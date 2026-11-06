package query

import (
	"bufio"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// SnapshotFormatVersion is the version of the snapshot schema. Bump it whenever the meaning of
// any field changes; snapshots with a different version are rejected.
const SnapshotFormatVersion = 1

// SnapshotHeader describes the conditions a snapshot was produced under.
type SnapshotHeader struct {
	FormatVersion         int32             `json:"format_version"`
	Revision              string            `json:"revision"`
	Dirty                 bool              `json:"dirty"`
	PlzVersion            string            `json:"plz_version"`
	Fingerprint           string            `json:"fingerprint"`
	FingerprintComponents map[string]string `json:"fingerprint_components"`
	ConfigHash            []byte            `json:"config_hash"`
	// PassEnv maps the name of every env var named in any target's pass_env to a hash of its value.
	PassEnv map[string]string `json:"pass_env"`
}

// TargetHashes is the per-target data DiffSnapshot needs from the "before" graph.
type TargetHashes struct {
	Rule  []byte `json:"rule_hash"`
	Tools []byte `json:"tool_path_hash,omitempty"` // nil if the target has no non-label tools
}

// GraphSnapshot is a compact, serialisable summary of a parsed build graph.
// It is keyed by the string form of each target's label.
type GraphSnapshot struct {
	Header  SnapshotHeader          `json:"header"`
	Targets map[string]TargetHashes `json:"targets"`
}

// NewSnapshot creates a snapshot of the given build state.
// The caller is expected to fill in the revision, dirty & fingerprint fields of the header.
func NewSnapshot(state *core.BuildState) *GraphSnapshot {
	targets := state.Graph.AllTargets()
	hashes := make([]TargetHashes, len(targets))
	parallelChunks(targets, func(start int, chunk []*core.BuildTarget) {
		for i, t := range chunk {
			hashes[start+i] = TargetHashes{
				Rule:  build.RuleHash(state, t, true, false),
				Tools: toolsHash(state, t),
			}
		}
	})
	s := &GraphSnapshot{
		Header: SnapshotHeader{
			FormatVersion: SnapshotFormatVersion,
			ConfigHash:    state.Hashes.Config,
			PassEnv:       passEnvHashes(targets),
		},
		Targets: make(map[string]TargetHashes, len(targets)),
	}
	for i, t := range targets {
		s.Targets[t.Label.String()] = hashes[i]
	}
	return s
}

// parallelChunks splits the targets into chunks and calls f on each one concurrently, passing it the
// index of the chunk's first target. It returns once they have all finished.
// f may write to disjoint parts of a shared slice indexed by target, but must otherwise not share state.
func parallelChunks(targets []*core.BuildTarget, f func(start int, chunk []*core.BuildTarget)) {
	size := max(len(targets)/runtime.NumCPU(), 100)
	var wg sync.WaitGroup
	for start := 0; start < len(targets); start += size {
		chunk := targets[start:min(start+size, len(targets))]
		wg.Go(func() { f(start, chunk) })
	}
	wg.Wait()
}

// passEnvHashes returns a hash of the current value of every env var named in any target's pass_env.
func passEnvHashes(targets []*core.BuildTarget) map[string]string {
	var m map[string]string
	for _, t := range targets {
		if t.PassEnv == nil {
			continue
		}
		for _, env := range *t.PassEnv {
			if _, present := m[env]; !present {
				if m == nil {
					m = map[string]string{}
				}
				m[env] = hashEnvValue(env)
			}
		}
	}
	return m
}

func hashEnvValue(name string) string {
	sum := sha256.Sum256([]byte(os.Getenv(name)))
	return hex.EncodeToString(sum[:])
}

// Fingerprint returns the compatibility key for a set of fingerprint components.
func Fingerprint(components map[string]string) string {
	keys := make([]string, 0, len(components))
	for k := range components {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, components[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// CheckHeader checks that the snapshot is compatible with the given fingerprint components.
// This only requires the config to be loaded, so can (and should) be done before parsing.
func (s *GraphSnapshot) CheckHeader(components map[string]string) error {
	if s.Header.FormatVersion != SnapshotFormatVersion {
		return fmt.Errorf("snapshot has format version %d, expected %d", s.Header.FormatVersion, SnapshotFormatVersion)
	}
	if fp := Fingerprint(components); s.Header.Fingerprint != fp {
		var diffs []string
		for k, v := range components {
			if theirs := s.Header.FingerprintComponents[k]; theirs != v {
				diffs = append(diffs, fmt.Sprintf("%s (snapshot: %q, current: %q)", k, theirs, v))
			}
		}
		for k, v := range s.Header.FingerprintComponents {
			if _, present := components[k]; !present {
				diffs = append(diffs, fmt.Sprintf("%s (snapshot: %q, current: unset)", k, v))
			}
		}
		sort.Strings(diffs)
		return fmt.Errorf("snapshot fingerprint %s doesn't match current fingerprint %s; differing components: %s", s.Header.Fingerprint, fp, strings.Join(diffs, ", "))
	}
	return nil
}

// CheckPassEnv checks that any env vars that are passed through to targets in the given (parsed)
// state have the same values as when the snapshot was produced. This can only be done after parsing.
func (s *GraphSnapshot) CheckPassEnv(state *core.BuildState) error {
	var diffs []string
	for name, hash := range passEnvHashes(state.Graph.AllTargets()) {
		// Vars that aren't in the snapshot can only be used by targets that are new or whose
		// pass_env has changed, both of which will be marked as changed anyway.
		if theirs, present := s.Header.PassEnv[name]; present && theirs != hash {
			diffs = append(diffs, name)
		}
	}
	if len(diffs) > 0 {
		sort.Strings(diffs)
		return fmt.Errorf("values of pass_env variables differ from the snapshot: %s", strings.Join(diffs, ", "))
	}
	return nil
}

// WriteSnapshotFile writes the snapshot to the given file, or stdout if it's empty or "-".
// The file is gzipped if its name ends in .gz, and written as JSON if asJSON is true.
func (s *GraphSnapshot) WriteSnapshotFile(filename string, asJSON bool) error {
	if filename == "" || filename == "-" {
		return s.write(os.Stdout, asJSON, false)
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	if err := s.write(f, asJSON, strings.HasSuffix(filename, ".gz")); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (s *GraphSnapshot) write(w io.Writer, asJSON, gzipped bool) error {
	if gzipped {
		gz := gzip.NewWriter(w)
		if err := s.write(gz, asJSON, false); err != nil {
			return err
		}
		return gz.Close()
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	}
	bw := bufio.NewWriter(w)
	if err := s.Write(bw); err != nil {
		return err
	}
	return bw.Flush()
}

// ReadSnapshotFile reads a snapshot from the given file, transparently handling gzip.
func ReadSnapshotFile(filename string) (*GraphSnapshot, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	if magic, _ := br.Peek(2); len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		return ReadSnapshot(gz)
	}
	return ReadSnapshot(br)
}

// Wire representation types for encoding/gob.
// Slices are used instead of maps to ensure deterministic serialization,
// as Go map iteration is randomized.
type wireSnapshot struct {
	Header  wireHeader
	Targets []targetEntry
}

type wireHeader struct {
	FormatVersion         int32
	Revision              string
	Dirty                 bool
	PlzVersion            string
	Fingerprint           string
	FingerprintComponents []kvEntry
	ConfigHash            []byte
	PassEnv               []kvEntry
}

type kvEntry struct {
	Key   string
	Value string
}

type targetEntry struct {
	Label  string
	Hashes TargetHashes
}

func (s *GraphSnapshot) toWire() wireSnapshot {
	w := wireSnapshot{
		Header: wireHeader{
			FormatVersion: s.Header.FormatVersion,
			Revision:      s.Header.Revision,
			Dirty:         s.Header.Dirty,
			PlzVersion:    s.Header.PlzVersion,
			Fingerprint:   s.Header.Fingerprint,
			ConfigHash:    s.Header.ConfigHash,
		},
		Targets: make([]targetEntry, 0, len(s.Targets)),
	}
	w.Header.FingerprintComponents = toSortedKV(s.Header.FingerprintComponents)
	w.Header.PassEnv = toSortedKV(s.Header.PassEnv)

	labels := make([]string, 0, len(s.Targets))
	for l := range s.Targets {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	for _, l := range labels {
		w.Targets = append(w.Targets, targetEntry{
			Label:  l,
			Hashes: s.Targets[l],
		})
	}
	return w
}

func toSortedKV(m map[string]string) []kvEntry {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]kvEntry, len(keys))
	for i, k := range keys {
		entries[i] = kvEntry{Key: k, Value: m[k]}
	}
	return entries
}

func fromKV(entries []kvEntry) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Key] = e.Value
	}
	return m
}

func fromWire(w *wireSnapshot) *GraphSnapshot {
	s := &GraphSnapshot{
		Header: SnapshotHeader{
			FormatVersion:         w.Header.FormatVersion,
			Revision:              w.Header.Revision,
			Dirty:                 w.Header.Dirty,
			PlzVersion:            w.Header.PlzVersion,
			Fingerprint:           w.Header.Fingerprint,
			FingerprintComponents: fromKV(w.Header.FingerprintComponents),
			ConfigHash:            w.Header.ConfigHash,
			PassEnv:               fromKV(w.Header.PassEnv),
		},
		Targets: make(map[string]TargetHashes, len(w.Targets)),
	}
	for _, t := range w.Targets {
		s.Targets[t.Label] = t.Hashes
	}
	return s
}

// Write writes the snapshot to the given writer as a binary gob GraphSnapshot message.
// Targets are written in sorted order so identical graphs produce identical files.
func (s *GraphSnapshot) Write(w io.Writer) error {
	ws := s.toWire()
	return gob.NewEncoder(w).Encode(&ws)
}

// ReadSnapshot reads a binary gob GraphSnapshot message from the given reader.
func ReadSnapshot(r io.Reader) (*GraphSnapshot, error) {
	var ws wireSnapshot
	if err := gob.NewDecoder(r).Decode(&ws); err != nil {
		return nil, fmt.Errorf("invalid snapshot: %w", err)
	}
	return fromWire(&ws), nil
}

// toolsHash hashes the resolved paths of a target's non-label tools. It returns nil if there are none.
// In-repo tools are skipped since they are handled via revdeps, and tools outside the repo
// shouldn't change, so hashing the resolved tool path is enough.
func toolsHash(state *core.BuildState, target *core.BuildTarget) []byte {
	var hash []byte
	for _, tool := range target.AllTools() {
		if _, ok := tool.Label(); ok {
			continue
		}
		h := sha1.New()
		for _, path := range tool.LocalPaths(state.Graph) {
			h.Write([]byte(path))
		}
		hash = h.Sum(hash)
	}
	return hash
}
