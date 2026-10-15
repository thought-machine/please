package query

import (
	"bytes"
	"encoding/gob"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

func TestSnapshotRoundTrip(t *testing.T) {
	s := core.NewDefaultBuildState()
	s.Hashes.Config = []byte{1, 2, 3}
	t1 := addTarget(s, "//src/core:core", nil, "src/core/core.go")
	t1.AddTool(core.SystemPathLabel{Name: "non-existent", Path: s.Config.Path()})
	t1.PassEnv = &[]string{"PATH"}
	t2 := addTarget(s, "///third_party/go/mod//:mod", t1)
	t2.Subrepo = core.NewSubrepo(s, "go_mod", "third_party/go", t1, cli.Arch{}, false)

	snapshot := NewSnapshot(s)
	snapshot.Header.Revision = "abc123"
	snapshot.Header.PlzVersion = "17.0.0"
	snapshot.Header.FingerprintComponents = map[string]string{"a": "b"}
	snapshot.Header.Fingerprint = Fingerprint(snapshot.Header.FingerprintComponents)
	assert.Len(t, snapshot.Targets, 2)
	assert.NotEmpty(t, snapshot.Targets["//src/core:core"].Tools)
	assert.Contains(t, snapshot.Header.PassEnv, "PATH")

	var buf bytes.Buffer
	require.NoError(t, snapshot.Write(&buf))
	read, err := ReadSnapshot(&buf)
	require.NoError(t, err)
	assert.Equal(t, snapshot, read)

	// And via a gzipped file.
	filename := filepath.Join(t.TempDir(), "snapshot.gz")
	require.NoError(t, snapshot.WriteSnapshotFile(filename, false))
	read, err = ReadSnapshotFile(filename)
	require.NoError(t, err)
	assert.Equal(t, snapshot, read)
}

func TestSnapshotIsDeterministic(t *testing.T) {
	s := core.NewDefaultBuildState()
	for _, l := range []string{"//a:a", "//b:b", "//c:c", "//d:d"} {
		addTarget(s, l, nil)
	}
	var buf1, buf2 bytes.Buffer
	require.NoError(t, NewSnapshot(s).Write(&buf1))
	require.NoError(t, NewSnapshot(s).Write(&buf2))
	assert.Equal(t, buf1.Bytes(), buf2.Bytes())
}

func TestReadSnapshotSkipsUnknownFields(t *testing.T) {
	s := core.NewDefaultBuildState()
	addTarget(s, "//src/core:core", nil, "src/core/core.go")
	snapshot := NewSnapshot(s)

	type futureWireSnapshot struct {
		Header     wireHeader
		Targets    []targetEntry
		ExtraField string
	}

	ws := snapshot.toWire()
	future := futureWireSnapshot{
		Header:     ws.Header,
		Targets:    ws.Targets,
		ExtraField: "from the future",
	}
	var buf bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buf).Encode(future))
	read, err := ReadSnapshot(&buf)
	require.NoError(t, err)
	assert.Equal(t, snapshot, read)
}

func TestReadSnapshotRejectsGarbage(t *testing.T) {
	_, err := ReadSnapshot(bytes.NewReader([]byte{0x0a, 0xff}))
	assert.Error(t, err)
}

func TestCheckHeader(t *testing.T) {
	components := map[string]string{"plz_version": "17.0.0", "target_arch": "linux_amd64"}
	snapshot := &GraphSnapshot{Header: SnapshotHeader{
		FormatVersion:         SnapshotFormatVersion,
		Fingerprint:           Fingerprint(components),
		FingerprintComponents: components,
	}}
	assert.NoError(t, snapshot.CheckHeader(components))

	err := snapshot.CheckHeader(map[string]string{"plz_version": "17.1.0", "target_arch": "linux_amd64"})
	assert.ErrorContains(t, err, "plz_version")
	assert.NotContains(t, err.Error(), "target_arch")

	snapshot.Header.FormatVersion = SnapshotFormatVersion + 1
	assert.ErrorContains(t, snapshot.CheckHeader(components), "format version")
}

func TestCheckPassEnv(t *testing.T) {
	t.Setenv("PLZ_SNAPSHOT_TEST_VAR", "before")
	s := core.NewDefaultBuildState()
	target := addTarget(s, "//src/core:core", nil)
	target.PassEnv = &[]string{"PLZ_SNAPSHOT_TEST_VAR"}
	snapshot := NewSnapshot(s)
	assert.NoError(t, snapshot.CheckPassEnv(s))

	t.Setenv("PLZ_SNAPSHOT_TEST_VAR", "after")
	assert.ErrorContains(t, snapshot.CheckPassEnv(s), "PLZ_SNAPSHOT_TEST_VAR")

	// Vars that weren't in the snapshot at all are fine; those targets will be marked changed anyway.
	delete(snapshot.Header.PassEnv, "PLZ_SNAPSHOT_TEST_VAR")
	assert.NoError(t, snapshot.CheckPassEnv(s))
}

func TestConfigChangeMarksEverythingChanged(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	s1.Hashes.Config = []byte{1}
	s2.Hashes.Config = []byte{2}
	addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s2, "//src/core:core", nil, "src/core/core.go")
	assert.EqualValues(t, []core.BuildLabel{t2.Label}, DiffSnapshot(roundTrip(t, NewSnapshot(s1)), s2, nil, 0, false))
}

// TestDiffSnapshotMatchesOracle checks that diffing against a serialised snapshot gives exactly
// the same result as the original graph-vs-graph implementation.
func TestDiffSnapshotMatchesOracle(t *testing.T) {
	for name, setup := range map[string]func(s1, s2 *core.BuildState){
		"unchanged": func(s1, s2 *core.BuildState) {
			addTarget(s1, "//src/core:core", nil, "src/core/core.go")
			addTarget(s2, "//src/core:core", nil, "src/core/core.go")
		},
		"source changed": func(s1, s2 *core.BuildState) {
			t1 := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
			addTarget(s1, "//src/query:changes", t1, "src/query/changes.go")
			t1 = addTarget(s2, "//src/core:core", nil, "src/core/core_changed.go")
			addTarget(s2, "//src/query:changes", t1, "src/query/changes.go")
		},
		"command changed": func(s1, s2 *core.BuildState) {
			addTarget(s1, "//src/core:core", nil).Command = "true"
			addTarget(s2, "//src/core:core", nil).Command = "false"
		},
		"new target": func(s1, s2 *core.BuildState) {
			addTarget(s1, "//src/core:core", nil)
			addTarget(s2, "//src/core:core", nil)
			addTarget(s2, "//src/core:new", nil)
		},
		"removed target": func(s1, s2 *core.BuildState) {
			addTarget(s1, "//src/core:core", nil)
			addTarget(s1, "//src/core:old", nil)
			addTarget(s2, "//src/core:core", nil)
		},
		"tool path changed": func(s1, s2 *core.BuildState) {
			addTarget(s1, "//src/core:core", nil).AddTool(core.SystemPathLabel{Name: "sh", Path: []string{"/bin"}})
			addTarget(s2, "//src/core:core", nil).AddTool(core.SystemPathLabel{Name: "sh", Path: []string{"/usr/bin"}})
		},
		"subrepo target changed": func(s1, s2 *core.BuildState) {
			for i, s := range []*core.BuildState{s1, s2} {
				t1 := addTarget(s, "//third_party/go:mod", nil)
				srcs := []string{}
				if i == 1 {
					srcs = []string{"test.go"}
				}
				t2 := addTarget(s, "///third_party/go/mod//:mod", nil, srcs...)
				t2.Subrepo = core.NewSubrepo(s, "go_mod", "third_party/go", t1, cli.Arch{}, false)
				addTarget(s, "//src/core:core", t2)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, level := range []int{0, 1, -1} {
				for _, includeSubrepos := range []bool{false, true} {
					s1 := core.NewDefaultBuildState()
					s2 := core.NewDefaultBuildState()
					setup(s1, s2)
					expected := changedTargets(s2, nil, oracleDiffGraphs(s1, s2), level, includeSubrepos)
					actual := DiffSnapshot(roundTrip(t, NewSnapshot(s1)), s2, nil, level, includeSubrepos)
					assert.Equal(t, expected, actual, "level %d, includeSubrepos %v", level, includeSubrepos)
				}
			}
		})
	}
}

func roundTrip(t *testing.T, s *GraphSnapshot) *GraphSnapshot {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, s.Write(&buf))
	s, err := ReadSnapshot(&buf)
	require.NoError(t, err)
	return s
}

// oracleDiffGraphs is the original graph-vs-graph implementation of diffGraphs, retained to
// check that snapshot-based diffing is equivalent.
func oracleDiffGraphs(before, after *core.BuildState) map[*core.BuildTarget]struct{} {
	configChanged := !bytes.Equal(before.Hashes.Config, after.Hashes.Config)
	changed := map[*core.BuildTarget]struct{}{}
	for _, afterTarget := range after.Graph.AllTargets() {
		beforeTarget := before.Graph.Target(afterTarget.Label)
		if beforeTarget == nil || configChanged ||
			!bytes.Equal(build.RuleHash(before, beforeTarget, true, false), build.RuleHash(after, afterTarget, true, false)) ||
			!bytes.Equal(toolsHash(before, beforeTarget), toolsHash(after, afterTarget)) {
			changed[afterTarget] = struct{}{}
		}
	}
	return changed
}
