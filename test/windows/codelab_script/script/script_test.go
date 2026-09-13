package script

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sidecarPath       = "test/windows/codelab_steps.conf"
	knownFailuresPath = "test/windows/codelab_known_failures.txt"
	censusPath        = "test/windows/codelab_script/script/test_data/census.txt"
)

// The real codelabs and the real sidecar, not samples that could quietly stop resembling them.
func loadReal(t *testing.T) ([]Codelab, *Sidecar) {
	t.Helper()
	files, err := filepath.Glob("docs/codelabs/*.md")
	require.NoError(t, err)
	require.Len(t, files, 8, "the codelabs as published at https://please.build/codelabs.html")
	sort.Strings(files)

	var codelabs []Codelab
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		codelabs = append(codelabs, ParseCodelab(f, string(b)))
	}
	b, err := os.ReadFile(sidecarPath)
	require.NoError(t, err)
	side, err := ParseSidecar(sidecarPath, string(b))
	require.NoError(t, err)
	return codelabs, side
}

// The drift guard. A codelab edit that introduces a block nothing can classify, or that changes
// a block the sidecar made a decision about, fails here on Linux rather than silently changing
// what the Windows job checks.
func TestEveryBlockIsDecided(t *testing.T) {
	codelabs, side := loadReal(t)
	_, errs := BuildPlan(codelabs, side)
	for _, err := range errs {
		t.Errorf("%s", err)
	}
}

func TestCensusMatchesGolden(t *testing.T) {
	codelabs, side := loadReal(t)
	want, err := os.ReadFile(censusPath)
	require.NoError(t, err)
	assert.Equal(t, string(want), Census(codelabs, side),
		"the census has changed; if the codelab edit that changed it is intended, regenerate %s with --format summary and review the diff", censusPath)
}

// The Linux half of the shrink-only rule: an entry naming a step that no longer exists fails
// here. The Windows half, an entry whose step passed, is in run_codelabs.ps1.
func TestKnownFailuresNameRealSteps(t *testing.T) {
	codelabs, side := loadReal(t)
	plan, errs := BuildPlan(codelabs, side)
	require.Empty(t, errs)

	names := map[string]bool{}
	runnable := map[string]bool{}
	for _, c := range plan.Codelabs {
		names[c.ID] = true
		for _, s := range c.Steps {
			names[s.Key] = true
			runnable[s.Key] = s.Kind == "run"
		}
	}
	b, err := os.ReadFile(knownFailuresPath)
	require.NoError(t, err)
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !names[line] {
			t.Errorf("%s:%d: %s names no codelab or step", knownFailuresPath, i+1, line)
		} else if strings.Contains(line, "::") && !runnable[line] {
			t.Errorf("%s:%d: %s is not a step that runs, so it can neither fail nor pass", knownFailuresPath, i+1, line)
		}
	}
}

func parseOne(t *testing.T, md string) Codelab {
	t.Helper()
	return ParseCodelab("test.md", "id: test\nsummary: Test\nstatus: Published\n\n"+md)
}

func mustSidecar(t *testing.T, conf string) *Sidecar {
	t.Helper()
	side, err := ParseSidecar("test.conf", conf)
	require.NoError(t, err)
	return side
}

func TestFrontMatterAndKeys(t *testing.T) {
	c := parseOne(t, "## Hello, world!\n```bash\nplz init\n```\n\n```bash\nplz build\n```\n## Next step\n```bash\nplz test\n```\n")
	assert.Equal(t, "test", c.ID)
	assert.Equal(t, "Test", c.Title)
	require.Len(t, c.Blocks, 3)
	assert.Equal(t, "test::hello-world/b1", Key(c, c.Blocks[0]))
	assert.Equal(t, "test::hello-world/b2", Key(c, c.Blocks[1]))
	// A new section restarts the ordinal, so an edit in one section renumbers no other.
	assert.Equal(t, "test::next-step/b1", Key(c, c.Blocks[2]))
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name, md string
		kind     Kind
		path     string
	}{
		{"file heading", "### `src/BUILD`\n```python\ngo_binary()\n```", KindFile, "src/BUILD"},
		{"file in prose", "Create a file `hello_service/service.go`:\n\n```golang\npackage main\n```", KindFile, "hello_service/service.go"},
		{"bare BUILD in prose", "Add a filegroup at `BUILD` in repo root:\n```python\nfilegroup()\n```", KindFile, "BUILD"},
		{"commands", "Run:\n```bash\nplz init\n```", KindCommand, ""},
		{"inline env prefix", "Like so:\n```bash\nGODEBUG=\"installgoroot=all\" go install std\n```", KindCommand, ""},
		{"transcript", "```\n$ plz build //:x\nBuild finished\n```", KindTranscript, ""},
		{"output in a bash fence", "The output should look like this:\n```bash\n.\n├── pleasew\n```", KindIllustration, ""},
		// Each of these is a line in the codelabs that would otherwise be a file to create.
		{"absolute path", "if Go is at `/opt/homebrew/bin/go`:\n```ini\n[Build]\n```", KindUnclassified, ""},
		{"module path", "Let's add `github.com/stretchr/testify`:\n```text\ngo_repo()\n```", KindUnclassified, ""},
		{"no rule", "By default:\n```\n/usr/local/bin:/usr/bin:/bin\n```", KindUnclassified, ""},
		// A sentence naming a file is often followed by the command that makes it.
		{"command beats prose", "Sync the changes to `third_party/go/BUILD`:\n```bash\nplz puku sync -w\n```", KindCommand, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := parseOne(t, "## S\n"+tc.md+"\n")
			require.Len(t, c.Blocks, 1)
			kind, path, err := Classify(c, c.Blocks[0], Key(c, c.Blocks[0]), mustSidecar(t, ""))
			require.NoError(t, err)
			assert.Equal(t, tc.kind, kind)
			assert.Equal(t, tc.path, path)
		})
	}
}

func TestUnclassifiedIsAnError(t *testing.T) {
	c := parseOne(t, "## S\nBy default:\n```\n/usr/local/bin\n```\n")
	_, errs := BuildPlan([]Codelab{c}, mustSidecar(t, ""))
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "test.md:7")
	assert.Contains(t, errs[0].Error(), "[test::s/b1]")
}

func TestCommandsSplitChainsIntoChdir(t *testing.T) {
	c := parseOne(t, "## S\n```bash\nmkdir x && cd x\nplz init\ntree -a\n```\n")
	plan, errs := BuildPlan([]Codelab{c}, mustSidecar(t, ""))
	require.Empty(t, errs)
	steps := plan.Codelabs[0].Steps
	require.Len(t, steps, 4)
	assert.Equal(t, Step{Key: "test::s/b1.1", Kind: "run", Line: 6, Section: "S", Command: "mkdir x"}, steps[0])
	assert.Equal(t, "chdir", steps[1].Kind)
	assert.Equal(t, "x", steps[1].Dir)
	assert.False(t, steps[2].NonBlocking)
	// Only displays something, so its failure blocks nothing after it.
	assert.True(t, steps[3].NonBlocking)
}

func TestTranscriptOutputIsAdvisory(t *testing.T) {
	c := parseOne(t, "## S\n```\n$ plz build //:x\n$ cat plz-out/gen/x\nhello\n```\n")
	plan, errs := BuildPlan([]Codelab{c}, mustSidecar(t, ""))
	require.Empty(t, errs)
	steps := plan.Codelabs[0].Steps
	require.Len(t, steps, 2)
	assert.Equal(t, "plz build //:x", steps[0].Command)
	assert.Empty(t, steps[0].ExpectedOutput)
	assert.Equal(t, []string{"hello"}, steps[1].ExpectedOutput)
	assert.Empty(t, steps[1].Assert)
}

func TestSidecarRequiresAReason(t *testing.T) {
	_, err := ParseSidecar("test.conf", "[test::s/b1]\nkind = illustration\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no reason")

	// A blank line between the reason and the stanza detaches it.
	_, err = ParseSidecar("test.conf", "; why\n\n[test::s/b1]\nkind = illustration\n")
	assert.Error(t, err)
}

func TestSidecarDecisions(t *testing.T) {
	c := parseOne(t, "## S\n### `.plzconfig`\n```\n[Plugin \"go\"]\n```\n\n```bash\nplz build\nplz-out/bin/main.pex\n```\n\n```bash\neval $(minikube docker-env)\n```\n")
	side := mustSidecar(t, `; a fragment
[test::s/b1]
matches = .plzconfig
mode = merge

; nothing after depends on it
[test::s/b2.2]
matches = plz-out/bin/main.pex
blocking = false

; bash only
[test::s/b3]
matches = eval $(minikube docker-env)
skip = unix-shell
`)
	plan, errs := BuildPlan([]Codelab{c}, side)
	require.Empty(t, errs)
	steps := plan.Codelabs[0].Steps
	require.Len(t, steps, 4)
	assert.Equal(t, "merge", steps[0].Mode)
	assert.False(t, steps[1].NonBlocking)
	assert.True(t, steps[2].NonBlocking)
	assert.Equal(t, "skip", steps[3].Kind)
	assert.Equal(t, "unix-shell", steps[3].Reason)
	assert.Equal(t, "bash only", steps[3].Detail)
}

func TestSidecarMatchesIsEnforced(t *testing.T) {
	c := parseOne(t, "## S\n```bash\nplz build //:new\n```\n")
	side := mustSidecar(t, "; decided about the old command\n[test::s/b1]\nmatches = plz build //:old\nskip = placeholder\n")
	_, errs := BuildPlan([]Codelab{c}, side)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), `"plz build //:old"`)
}

func TestSidecarStanzaNamingNothingIsAnError(t *testing.T) {
	c := parseOne(t, "## S\n```bash\nplz build\n```\n")
	side := mustSidecar(t, "; about a block since deleted\n[test::s/b9]\nkind = illustration\n")
	_, errs := BuildPlan([]Codelab{c}, side)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "test::s/b9 names no block")
}

func TestNotRunnableCodelab(t *testing.T) {
	c := parseOne(t, "## S\n```yaml\nname: CI\n```\n")
	side := mustSidecar(t, "; all YAML\n[test]\nnot-runnable = no local commands\n")
	plan, errs := BuildPlan([]Codelab{c}, side)
	require.Empty(t, errs)
	assert.Equal(t, "no local commands", plan.Codelabs[0].NotRunnable)
	assert.Empty(t, plan.Codelabs[0].Steps)
	assert.Equal(t, 1, plan.Codelabs[0].Blocks["illustration"])
}
