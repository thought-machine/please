// Tests the command replacement functionality.

package build

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

var wd string
var state *core.BuildState

func init() {
	state = core.NewDefaultBuildState()
	wd, _ = os.Getwd()
}

func TestLocation(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target1 := makeTarget("//path/to:target1", "ln -s $(location //path/to:target2) ${OUT}", target2)

	expected := "ln -s path/to/target2.py ${OUT}"
	cmd := replaceSequences(state, target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestLocations(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.AddOutput("target2_other.py")
	target1 := makeTarget("//path/to:target1", "cat $(locations //path/to:target2) > ${OUT}", target2)

	expected := "cat path/to/target2.py path/to/target2_other.py > ${OUT}"
	cmd := replaceSequences(state, target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestExe(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.IsBinary = true
	target1 := makeTarget("//path/to:target1", "$(exe //path/to:target2) -o ${OUT}", target2)

	expected := "path/to/target2.py -o ${OUT}"
	cmd := replaceSequences(state, target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestOutExe(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.IsBinary = true
	target1 := makeTarget("//path/to:target1", "$(out_exe //path/to:target2) -o ${OUT}", target2)

	expected := "plz-out/bin/path/to/target2.py -o ${OUT}"
	cmd := replaceSequences(state, target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestJavaExe(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.IsBinary = true
	target2.AddLabel("java_non_exe") // This label tells us to prefix it with java -jar.
	target1 := makeTarget("//path/to:target1", "$(exe //path/to:target2) -o ${OUT}", target2)

	expected := "java -jar path/to/target2.py -o ${OUT}"
	cmd := replaceSequences(state, target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestJavaOutExe(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.IsBinary = true
	target2.AddLabel("java_non_exe") // This label tells us to prefix it with java -jar.
	target1 := makeTarget("//path/to:target1", "$(out_exe //path/to:target2) -o ${OUT}", target2)

	expected := "java -jar plz-out/bin/path/to/target2.py -o ${OUT}"
	cmd := replaceSequences(state, target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestReplacementsForTest(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target1 := makeTarget("//path/to:target1", "$(exe //path/to:target1) $(location //path/to:target2)", target2)
	target1.IsBinary = true
	target1.IsTest = true

	expected := "./target1.py path/to/target2.py"
	cmd := ReplaceTestSequences(state, target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestDataReplacementForTest(t *testing.T) {
	target := makeTarget("//path/to:target1", "cat $(location test_data.txt)", nil)
	target.Data = append(target.Data, core.FileLabel{File: "test_data.txt", Package: "path/to"})

	expected := "cat path/to/test_data.txt"
	cmd := ReplaceTestSequences(state, target, target.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestAmpersandReplacement(t *testing.T) {
	target := makeTarget("//path/to:target1", "cat $(location b&b.txt)", nil)
	expected := "cat \"path/to/b&b.txt\""
	cmd := ReplaceSequences(state, target, target.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestToolReplacement(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "blah", nil)
	target1 := makeTarget("//path/to:target1", "$(location //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	wd, _ := os.Getwd()
	expected := quote(path.Join(wd, "plz-out/gen/path/to/target2.py"))
	cmd := ReplaceSequences(state, target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestDirReplacement(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "blah", nil)
	target2.AddOutput("blah2.txt")
	target1 := makeTarget("//path/to:target1", "$(dir //path/to:target2)", target2)

	expected := "path/to"
	cmd := ReplaceSequences(state, target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestToolDirReplacement(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "blah", nil)
	target2.AddOutput("blah2.txt")
	target1 := makeTarget("//path/to:target1", "$(dir //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	wd, _ := os.Getwd()
	expected := quote(path.Join(wd, "plz-out/gen/path/to"))
	cmd := ReplaceSequences(state, target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestBazelCompatReplacements(t *testing.T) {
	// Check that we don't do any of these things normally.
	target := makeTarget("//path/to:target", "cp $< $@", nil)
	assert.Equal(t, "cp $< $@", replaceSequences(state, target))
	// In Bazel compat mode we do though.
	state := core.NewBuildState(1, nil, 1, core.DefaultConfiguration())
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
	err = ioutil.WriteFile("plz-out/gen/path/to/target2.py", []byte(`"""Test file for command_replacements_test"""`), 0644)
	assert.NoError(t, err)

	target2 := makeTarget("//path/to:target2", "cp $< $@", nil)
	target := makeTarget("//path/to:target", "echo $(hash //path/to:target2)", target2)
	assert.Panics(t, func() { replaceSequences(state, target) }, "Can't use $(hash ) on a non-stamped target")
	target.Stamp = true
	// Note that this hash is determined arbitrarily, it doesn't matter for this test
	// precisely what its value is.
	assert.Equal(t, "echo gB4sUwsLkB1ODYKUxYrKGlpdYUI", replaceSequences(state, target))
}

func TestWorkerReplacement(t *testing.T) {
	tool := makeTarget("//path/to:target2", "", nil)
	tool.IsBinary = true
	target := makeTarget("//path/to:target", "$(worker //path/to:target2) --some_arg", tool)
	target.Tools = append(target.Tools, tool.Label)
	worker, remoteArgs, localCmd := workerCommandAndArgs(state, target)
	assert.Equal(t, wd+"/plz-out/bin/path/to/target2.py", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "", localCmd)
}

func TestSystemWorkerReplacement(t *testing.T) {
	target := makeTarget("//path/to:target", "$(worker /usr/bin/javac) --some_arg", nil)
	target.Tools = append(target.Tools, core.SystemFileLabel{Path: "/usr/bin/javac"})
	worker, remoteArgs, localCmd := workerCommandAndArgs(state, target)
	assert.Equal(t, "/usr/bin/javac", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "", localCmd)
}

func TestLocalCommandWorker(t *testing.T) {
	tool := makeTarget("//path/to:target2", "", nil)
	tool.IsBinary = true
	target := makeTarget("//path/to:target", "$(worker //path/to:target2) --some_arg && find . | xargs rm && echo hello", tool)
	target.Tools = append(target.Tools, tool.Label)
	worker, remoteArgs, localCmd := workerCommandAndArgs(state, target)
	assert.Equal(t, wd+"/plz-out/bin/path/to/target2.py", worker)
	assert.Equal(t, "--some_arg", remoteArgs)
	assert.Equal(t, "find . | xargs rm && echo hello", localCmd)
}

func TestWorkerCommandAndArgsMustComeFirst(t *testing.T) {
	tool := makeTarget("//path/to:target2", "", nil)
	tool.IsBinary = true
	target := makeTarget("//path/to:target", "something something $(worker javac)", tool)
	target.Tools = append(target.Tools, tool.Label)
	assert.Panics(t, func() { workerCommandAndArgs(state, target) })
}

func TestWorkerReplacementWithNoWorker(t *testing.T) {
	target := makeTarget("//path/to:target", "echo hello", nil)
	worker, remoteArgs, localCmd := workerCommandAndArgs(state, target)
	assert.Equal(t, "", worker)
	assert.Equal(t, "", remoteArgs)
	assert.Equal(t, "echo hello", localCmd)
}

func makeTarget(name string, command string, dep *core.BuildTarget) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(name, ""))
	target.Command = command
	target.AddOutput(target.Label.Name + ".py")
	if dep != nil {
		target.AddDependency(dep.Label)
		// This is a bit awkward but I don't want to add a public interface just for a test.
		graph := core.NewGraph()
		graph.AddTarget(target)
		graph.AddTarget(dep)
		graph.AddDependency(target.Label, dep.Label)
	}
	return target
}

func replaceSequences(state *core.BuildState, target *core.BuildTarget) string {
	return ReplaceSequences(state, target, target.GetCommand(state))
}
