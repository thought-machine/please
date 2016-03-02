// Tests the command replacement functionality.

package build

import (
	"os"
	"path"
	"testing"

	"core"
)

func TestLocation(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target1 := makeTarget("//path/to:target1", "ln -s $(location //path/to:target2) ${OUT}", target2)

	expected := "ln -s path/to/target2.py ${OUT}"
	cmd := replaceSequences(target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestLocations(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.AddOutput("target2_other.py")
	target1 := makeTarget("//path/to:target1", "cat $(locations //path/to:target2) > ${OUT}", target2)

	expected := "cat path/to/target2.py path/to/target2_other.py > ${OUT}"
	cmd := replaceSequences(target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestExe(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "", nil)
	target2.IsBinary = true
	target1 := makeTarget("//path/to:target1", "$(exe //path/to:target2) -o ${OUT}", target2)

	expected := "path/to/target2.py -o ${OUT}"
	cmd := replaceSequences(target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestLocationPairs(t *testing.T) {
	core.RepoRoot = "/home/user/repo"
	target2 := makeTarget("//path/to:target2", "", nil)
	target1 := makeTarget("//path/to:target1", "$(location_pairs //path/to:target2)", target2)
	expected := "/home/user/repo/plz-out/gen/path/to/target2.py target2.py"
	cmd := replaceSequences(target1)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestLocationPairsLocalFile(t *testing.T) {
	core.RepoRoot = "/home/user/repo"
	target1 := makeTarget("//path/to:target1", "$(location_pairs target1.py)", nil)
	expected := "/home/user/repo/path/to/target1.py target1.py"
	cmd := replaceSequences(target1)
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
	cmd := ReplaceTestSequences(target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestDataReplacementForTest(t *testing.T) {
	target := makeTarget("//path/to:target1", "cat $(location test_data.txt)", nil)
	target.Data = append(target.Data, core.FileLabel{File: "test_data.txt", Package: "path/to"})

	expected := "cat path/to/test_data.txt"
	cmd := ReplaceTestSequences(target, target.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestAmpersandReplacement(t *testing.T) {
	target := makeTarget("//path/to:target1", "cat $(location b&b.txt)", nil)
	expected := "cat \"path/to/b&b.txt\""
	cmd := ReplaceSequences(target, target.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestToolReplacement(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "blah", nil)
	target1 := makeTarget("//path/to:target1", "$(location //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	expected := quote(path.Join(os.Getenv("TEST_DIR"), "plz-out/gen/path/to/target2.py"))
	cmd := ReplaceSequences(target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestDirReplacement(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "blah", nil)
	target1 := makeTarget("//path/to:target1", "$(dir //path/to:target2)", target2)

	expected := "path/to"
	cmd := ReplaceSequences(target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
}

func TestToolDirReplacement(t *testing.T) {
	target2 := makeTarget("//path/to:target2", "blah", nil)
	target1 := makeTarget("//path/to:target1", "$(dir //path/to:target2)", target2)
	target1.Tools = append(target1.Tools, target2.Label)

	expected := quote(path.Join(os.Getenv("TEST_DIR"), "plz-out/gen/path/to"))
	cmd := ReplaceSequences(target1, target1.Command)
	if cmd != expected {
		t.Errorf("Replacement sequence not as expected; is %s, should be %s", cmd, expected)
	}
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
