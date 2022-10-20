// Tests the command replacement functionality.

package core

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var wd string
var state *BuildState

const testHash = "gB4sUwsLkB1ODYKUxYrKGlpdYUI"

func init() {
	state = NewDefaultBuildState()
	state.TargetHasher = &testHasher{}
	wd, _ = os.Getwd()
}

func TestLocation(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target1 := makeTarget2("//path/to:target1", "ln -s $(location //path/to:target2) ${OUT}", target2)

	expected := "ln -s path/to/target2.py ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestLocations(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target2.AddOutput("target2_other.py")
	target1 := makeTarget2("//path/to:target1", "cat $(locations //path/to:target2) > ${OUT}", target2)

	expected := "cat path/to/target2.py path/to/target2_other.py > ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestOutLocation(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target1 := makeTarget2("//path/to:target1", "ln -s $(out_location //path/to:target2) ${OUT}", target2)

	expected := "ln -s plz-out/gen/path/to/target2.py ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestOutLocations(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target2.AddOutput("target2_other.py")
	target1 := makeTarget2("//path/to:target1", "ln -s $(out_locations //path/to:target2) ${OUT}", target2)

	expected := "ln -s plz-out/gen/path/to/target2.py plz-out/gen/path/to/target2_other.py ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestExe(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target2.IsBinary = true
	target1 := makeTarget2("//path/to:target1", "$(exe //path/to:target2) -o ${OUT}", target2)

	expected := "path/to/target2.py -o ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestOutExe(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target2.IsBinary = true
	target1 := makeTarget2("//path/to:target1", "$(out_exe //path/to:target2) -o ${OUT}", target2)

	expected := "plz-out/bin/path/to/target2.py -o ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestJavaExe(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target2.IsBinary = true
	target2.AddLabel("java_non_exe") // This label tells us to prefix it with java -jar.
	target1 := makeTarget2("//path/to:target1", "$(exe //path/to:target2) -o ${OUT}", target2)

	expected := "java -jar path/to/target2.py -o ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestJavaOutExe(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target2.IsBinary = true
	target2.AddLabel("java_non_exe") // This label tells us to prefix it with java -jar.
	target1 := makeTarget2("//path/to:target1", "$(out_exe //path/to:target2) -o ${OUT}", target2)

	expected := "java -jar plz-out/bin/path/to/target2.py -o ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestReplacementsForTest(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "", nil)
	target1 := makeTarget2("//path/to:target1", "$(exe //path/to:target1) $(location //path/to:target2)", target2)
	target1.IsBinary = true
	target1.Test = new(TestFields)

	expected := "./target1.py path/to/target2.py"
	cmd, _ := ReplaceTestSequences(state, target1, target1.Command)
	assert.Equal(t, expected, cmd)
}

func TestDataReplacementForTest(t *testing.T) {
	target := makeTarget2("//path/to:target1", "cat $(location test_data.txt)", nil)
	target.AddDatum(FileLabel{File: "test_data.txt", Package: "path/to"})

	expected := "cat path/to/test_data.txt"
	cmd, _ := ReplaceTestSequences(state, target, target.Command)
	assert.Equal(t, expected, cmd)
}

func TestAmpersandReplacement(t *testing.T) {
	target := makeTarget2("//path/to:target1", "cat $(location b&b.txt)", nil)
	expected := "cat \"path/to/b&b.txt\""
	cmd, _ := ReplaceSequences(state, target, target.Command)
	assert.Equal(t, expected, cmd)
}

func TestToolReplacement(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "blah", nil)
	target1 := makeTarget2("//path/to:target1", "$(location //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	wd, _ := os.Getwd()
	expected := quote(filepath.Join(wd, "plz-out/gen/path/to/target2.py"))
	cmd, _ := ReplaceSequences(state, target1, target1.Command)
	assert.Equal(t, expected, cmd)
}

func TestToolReplacementSubrepo(t *testing.T) {
	target2 := makeTarget2("///subrepo//path/to:target2", "blah", nil)
	target1 := makeTarget2("///subrepo//path/to:target1", "$(location //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	wd, _ := os.Getwd()
	expected := quote(filepath.Join(wd, "plz-out/gen/subrepo/path/to/target2.py"))
	cmd, _ := ReplaceSequences(state, target1, target1.Command)
	assert.Equal(t, expected, cmd)
}

func TestDirReplacement(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "blah", nil)
	target2.AddOutput("blah2.txt")
	target1 := makeTarget2("//path/to:target1", "$(dir //path/to:target2)", target2)

	expected := "path/to"
	cmd, _ := ReplaceSequences(state, target1, target1.Command)
	assert.Equal(t, expected, cmd)
}

func TestOutDirReplacement(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "blah", nil)
	target2.AddOutput("blah2.txt")
	target1 := makeTarget2("//path/to:target1", "$(out_dir //path/to:target2)", target2)

	expected := "plz-out/gen/path/to"
	cmd, _ := ReplaceSequences(state, target1, target1.Command)
	assert.Equal(t, expected, cmd)
}

func TestToolDirReplacement(t *testing.T) {
	target2 := makeTarget2("//path/to:target2", "blah", nil)
	target2.AddOutput("blah2.txt")
	target1 := makeTarget2("//path/to:target1", "$(dir //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	wd, _ := os.Getwd()
	expected := quote(filepath.Join(wd, "plz-out/gen/path/to"))
	cmd, _ := ReplaceSequences(state, target1, target1.Command)
	assert.Equal(t, expected, cmd)
}

func TestBazelCompatReplacements(t *testing.T) {
	// Check that we don't do any of these things normally.
	target := makeTarget2("//path/to:target", "cp $< $@", nil)
	assert.Equal(t, "cp $< $@", replaceSequences(state, target))
	// In Bazel compat mode we do though.
	state := NewDefaultBuildState()
	state.Config.Bazel.Compatibility = true
	assert.Equal(t, "cp $SRCS $OUTS", replaceSequences(state, target))
	// @D is the output dir, for us it's the tmp dir.
	target.Command = "cp $SRCS $@D"
	assert.Equal(t, "cp $SRCS $TMP_DIR", replaceSequences(state, target))
	// This parenthesised syntax seems to be allowed too.
	target.Command = "cp $(<) $(@)"
	assert.Equal(t, "cp $SRCS $OUTS", replaceSequences(state, target))
}

func TestHashReplacement(t *testing.T) {
	// Need to write the file that will be used to calculate the hash.
	err := os.MkdirAll("plz-out/gen/path/to", 0755)
	assert.NoError(t, err)
	err = os.WriteFile("plz-out/gen/path/to/target2.py", []byte(`"""Test file for command_replacements_test"""`), 0644)
	assert.NoError(t, err)

	target2 := makeTarget2("//path/to:target2", "cp $< $@", nil)
	target := makeTarget2("//path/to:target", "echo $(hash //path/to:target2)", target2)
	assert.Equal(t, "echo "+testHash, replaceSequences(state, target))
}

func TestWorkerReplacement(t *testing.T) {
	tool := makeTarget2("//path/to:target2", "", nil)
	tool.IsBinary = true
	target := makeTarget2("//path/to:target", "$(worker //path/to:target2) --some_arg", tool)
	target.Tools = append(target.Tools, tool.Label)
	worker, remoteArgs, localCmd, err := WorkerCommandAndArgs(state, target)
	assert.NoError(t, err)
	assert.Equal(t, wd+"/plz-out/bin/path/to/target2.py", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "", localCmd)
}

func TestSystemWorkerReplacement(t *testing.T) {
	target := makeTarget2("//path/to:target", "$(worker /usr/bin/javac) --some_arg", nil)
	target.Tools = append(target.Tools, SystemFileLabel{Path: "/usr/bin/javac"})
	worker, remoteArgs, localCmd, err := WorkerCommandAndArgs(state, target)
	assert.NoError(t, err)
	assert.Equal(t, "/usr/bin/javac", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "", localCmd)
}

func TestLocalCommandWorker(t *testing.T) {
	tool := makeTarget2("//path/to:target2", "", nil)
	tool.IsBinary = true
	target := makeTarget2("//path/to:target", "$(worker //path/to:target2) --some_arg && find . | xargs rm && echo hello", tool)
	target.Tools = append(target.Tools, tool.Label)
	worker, remoteArgs, localCmd, err := WorkerCommandAndArgs(state, target)
	assert.NoError(t, err)
	assert.Equal(t, wd+"/plz-out/bin/path/to/target2.py", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "find . | xargs rm && echo hello", localCmd)
}

func TestWorkerCommandAndArgsMustComeFirst(t *testing.T) {
	tool := makeTarget2("//path/to:target2", "", nil)
	tool.IsBinary = true
	target := makeTarget2("//path/to:target", "something something $(worker javac)", tool)
	target.Tools = append(target.Tools, tool.Label)
	assert.Panics(t, func() { WorkerCommandAndArgs(state, target) })
}

func TestWorkerReplacementWithNoWorker(t *testing.T) {
	target := makeTarget2("//path/to:target", "echo hello", nil)
	worker, remoteArgs, localCmd, err := WorkerCommandAndArgs(state, target)
	assert.NoError(t, err)
	assert.Equal(t, "", worker)
	assert.Equal(t, "", remoteArgs)
	assert.Equal(t, "echo hello", localCmd)
}

func TestWorkerReplacementNotTarget(t *testing.T) {
	target := makeTarget2("//path/to:target", "$(worker javac_worker) --some_arg && find . | xargs rm && echo hello", nil)
	worker, remoteArgs, localCmd, err := WorkerCommandAndArgs(state, target)
	assert.NoError(t, err)
	assert.Equal(t, "javac_worker", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "find . | xargs rm && echo hello", localCmd)
}

func TestCrossCompileReplacement(t *testing.T) {
	target2 := makeTarget2("///linux_x86//path/to:target2", "", nil)
	target1 := makeTarget2("///linux_x86//path/to:target1", "ln -s $(location //path/to:target2) ${OUT}", target2)

	expected := "ln -s path/to/target2.py ${OUT}"
	assert.Equal(t, expected, replaceSequences(state, target1))
}

func TestEntryPoints(t *testing.T) {
	target := NewBuildTarget(ParseBuildLabel("//tools:foo", ""))
	target.EntryPoints = map[string]string{"some_ep": "bin/some_ep"}
	target.IsBinary = true
	target.AddOutput("bin/some_ep")
	graph := NewGraph()
	graph.AddTarget(target)

	cmd, err := ReplaceSequences(state, target, "$(out_exe //tools:foo|some_ep)")
	require.NoError(t, err)

	require.Equal(t, "plz-out/bin/tools/bin/some_ep", cmd)
}

func makeTarget2(name string, command string, dep *BuildTarget) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(name, ""))
	target.Command = command
	target.AddOutput(target.Label.Name + ".py")
	if dep != nil {
		target.AddDependency(dep.Label)
		// This is a bit awkward but I don't want to add a public interface just for a test.
		graph := NewGraph()
		graph.AddTarget(target)
		graph.AddTarget(dep)
		target.AddDependency(dep.Label)
		if err := target.ResolveDependencies(graph); err != nil {
			log.Fatalf("Failed to resolve some dependencies for %s: %s", target, err)
		}
	}
	return target
}

func replaceSequences(state *BuildState, target *BuildTarget) string {
	cmd, _ := ReplaceSequences(state, target, target.GetCommand(state))
	return cmd
}

type testHasher struct{}

func (h *testHasher) OutputHash(target *BuildTarget) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(testHash)
}

func (h *testHasher) SetHash(target *BuildTarget, hash []byte) {
}
