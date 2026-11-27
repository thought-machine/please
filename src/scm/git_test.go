package scm

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseChangedLines(t *testing.T) {
	b, err := os.ReadFile("src/scm/test_data/git.diff")
	assert.NoError(t, err)
	g := git{}
	m, err := g.parseChangedLines(b)
	assert.NoError(t, err)
	assert.Equal(t, map[string][]int{
		"test/python_rules/behave/BUILD":                                      {8},
		"test/python_rules/behave/features/behave_test3.feature":              {1, 10},
		"test/python_rules/behave/features/steps/behave_test_steps.py":        {24, 25, 26, 27, 28},
		"test/python_rules/behave/features/test_suite_1/behave_test1.feature": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		"test/python_rules/behave/features/test_suite_2/behave_test2.feature": {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		"tools/please_pex/behave.py":                                          {2, 3, 10, 11, 12, 13, 14, 15, 16, 17, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}, m)
}

func TestChangedFileNames(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	run := func(args ...string) {
		t.Helper()
		args = append([]string{"-c", "user.name=test", "-c", "user.email=test@please.build", "-c", "commit.gpgsign=false"}, args...)
		out, err := exec.Command("git", args...).CombinedOutput()
		assert.NoError(t, err, string(out))
	}
	write := func(name, contents string) {
		t.Helper()
		assert.NoError(t, os.WriteFile(name, []byte(contents), 0644))
	}

	run("init", "-q")
	run("config", "core.quotePath", "true")
	write("plain.txt", "1")
	write("caf\u00e9.txt", "1")
	run("add", ".")
	run("commit", "-q", "-m", "first")
	g := &git{repoRoot: dir}
	assert.Empty(t, g.ChangedFiles("", true, ""))

	write("plain.txt", "2")
	write("caf\u00e9.txt", "2")
	write("\u65b0.txt", "1")
	assert.ElementsMatch(t, []string{"caf\u00e9.txt", "plain.txt"}, g.ChangedFiles("", false, ""))
	assert.ElementsMatch(t, []string{"caf\u00e9.txt", "plain.txt", "\u65b0.txt"}, g.ChangedFiles("", true, ""))

	run("add", ".")
	run("commit", "-q", "-m", "second")
	expected := []string{"caf\u00e9.txt", "plain.txt", "\u65b0.txt"}
	assert.ElementsMatch(t, expected, g.ChangedFiles("HEAD~1", true, ""))
	assert.ElementsMatch(t, expected, g.ChangesIn("HEAD", ""))
}
