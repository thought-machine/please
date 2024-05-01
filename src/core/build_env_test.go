package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceEnvironment(t *testing.T) {
	env := BuildEnv{
		"TMP_DIR": "/home/user/please/src/core",
		"PKG": "src/core",
		"SRCS": "core.go build_env.go",
	}
	assert.Equal(t,
		"/home/user/please/src/core src/core core.go build_env.go",
		os.Expand("$TMP_DIR ${PKG} ${SRCS}", env.ReplaceEnvironment))
	assert.Equal(t, "", os.Expand("$WIBBLE", env.ReplaceEnvironment))
}

func TestReplace(t *testing.T) {
	env := BuildEnv{
		"TMP_DIR": "/home/user/please/src/core",
		"PKG": "src/core",
		"SRCS": "core.go build_env.go",
	}
	env.Replace("PKG", "src/test")
	assert.EqualValues(t, BuildEnv{
		"TMP_DIR": "/home/user/please/src/core",
		"PKG": "src/test",
		"SRCS": "core.go build_env.go",
	}, env)
}

func TestRedact(t *testing.T) {
	env := BuildEnv{
		"WHATEVER": "12345",
		"GPG_PASSWORD": "54321",
		"ULTIMATE_MEGASECRET": "42",
	}
	expected := BuildEnv{
		"WHATEVER": "12345",
		"GPG_PASSWORD": "************",
		"ULTIMATE_MEGASECRET": "************",
	}
	assert.EqualValues(t, expected, env.Redacted())
}

func TestString(t *testing.T) {
	env := BuildEnv{
		"A": "B",
		"C": "D",
	}
	assert.EqualValues(t, "A=B\nC=D", env.String())
}

func TestExecEnvironment(t *testing.T) {
	t.Setenv("TERM", "my-term")

	// Set up target.
	target := NewBuildTarget(NewBuildLabel("pkg", "t"))
	target.AddOutput("out_file1")
	target.AddDatum(FileLabel{File: "data_file1", Package: "pkg"})

	env := ExecEnvironment(NewDefaultBuildState(), target, "/path/to/runtime/dir")

	assert.Equal(t, env["DATA"], "pkg/data_file1")
	assert.Equal(t, env["TMP_DIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["TMPDIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["HOME"], "/path/to/runtime/dir")
	assert.Equal(t, env["TERM"], "my-term")
	assert.Equal(t, env["OUTS"], "out_file1")
	assert.Equal(t, env["OUT"], "out_file1")
	assert.NotContains(t, env, "TEST")
}

func TestExecEnvironmentTestTarget(t *testing.T) {
	t.Setenv("TERM", "my-term")

	state := NewDefaultBuildState()

	rootPkg := NewPackage("")
	// Set up tool 1.
	tool1 := NewBuildTarget(NewBuildLabel("", "tool1"))
	tool1.IsBinary = true
	tool1.AddOutput("tool1")
	state.Graph.AddTarget(tool1)
	// Set up tool 2.
	tool2 := NewBuildTarget(NewBuildLabel("", "tool2"))
	tool2.IsBinary = true
	tool2.AddOutput("tool2")
	state.Graph.AddTarget(tool2)
	state.Graph.AddPackage(rootPkg)

	// Set up test target.
	testTarget := NewBuildTarget(NewBuildLabel("pkg", "t"))
	testTarget.Test = new(TestFields)
	testTarget.AddOutput("out_file1")
	testTarget.AddDatum(FileLabel{File: "data_file1", Package: "pkg"})
	testTarget.AddNamedDatum("file2", FileLabel{File: "data_file2", Package: "pkg"})
	testTarget.AddTestTool(tool1.Label)
	testTarget.AddNamedTestTool("tool2", tool2.Label)

	env := ExecEnvironment(state, testTarget, "/path/to/runtime/dir")

	assert.Equal(t, env["DATA"], "pkg/data_file1 pkg/data_file2")
	assert.Equal(t, env["DATA_FILE2"], "pkg/data_file2")
	assert.Equal(t, env["TOOLS"], "plz-out/bin/tool1 plz-out/bin/tool2")
	assert.Equal(t, env["TOOLS_TOOL2"], "plz-out/bin/tool2")
	assert.Equal(t, env["TMP_DIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["TMPDIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["HOME"], "/path/to/runtime/dir")
	assert.Equal(t, env["TERM"], "my-term")
	assert.Equal(t, env["OUTS"], "out_file1")
	assert.Equal(t, env["OUT"], "out_file1")
	assert.Equal(t, env["TEST"], "out_file1")
}

func TestExecEnvironmentDebugTarget(t *testing.T) {
	t.Setenv("TERM", "my-term")

	state := NewDefaultBuildState()

	// Set up tool 1.
	rootPkg := NewPackage("")
	tool1 := NewBuildTarget(NewBuildLabel("", "tool1"))
	tool1.IsBinary = true
	tool1.AddOutput("tool1")
	state.Graph.AddTarget(tool1)
	state.Graph.AddPackage(rootPkg)

	// Set up debug target.
	target := NewBuildTarget(NewBuildLabel("pkg", "t"))
	target.AddOutput("out_file1")
	target.AddDebugDatum(FileLabel{File: "data_file1", Package: "pkg"})
	target.AddNamedDebugTool("tool1", tool1.Label)

	env := ExecEnvironment(state, target, "/path/to/runtime/dir")

	assert.Equal(t, env["DEBUG_DATA"], "pkg/data_file1")
	assert.Equal(t, env["DEBUG_TOOLS"], "plz-out/bin/tool1")
	assert.Equal(t, env["DEBUG_TOOLS_TOOL1"], "plz-out/bin/tool1")
	assert.Equal(t, env["DEBUG_TOOL"], "plz-out/bin/tool1")
	assert.Equal(t, env["TMP_DIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["TMPDIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["HOME"], "/path/to/runtime/dir")
	assert.Equal(t, env["TERM"], "my-term")
	assert.Equal(t, env["OUTS"], "out_file1")
	assert.Equal(t, env["OUT"], "out_file1")
	assert.NotContains(t, env, "TEST")
}

func TestExecEnvironmentDebugTestTarget(t *testing.T) {
	t.Setenv("TERM", "my-term")

	state := NewDefaultBuildState()

	// Set up tool 1.
	rootPkg := NewPackage("")
	tool1 := NewBuildTarget(NewBuildLabel("", "tool1"))
	tool1.IsBinary = true
	tool1.AddOutput("tool1")
	state.Graph.AddTarget(tool1)
	state.Graph.AddPackage(rootPkg)

	// Set up debug target.
	testTarget := NewBuildTarget(NewBuildLabel("pkg", "t"))
	testTarget.Test = new(TestFields)
	testTarget.AddOutput("out_file1")
	testTarget.AddDebugDatum(FileLabel{File: "data_file1", Package: "pkg"})
	testTarget.AddNamedDebugTool("tool1", tool1.Label)

	env := ExecEnvironment(state, testTarget, "/path/to/runtime/dir")

	assert.Equal(t, env["DEBUG_DATA"], "pkg/data_file1")
	assert.Equal(t, env["DEBUG_TOOLS"], "plz-out/bin/tool1")
	assert.Equal(t, env["DEBUG_TOOLS_TOOL1"], "plz-out/bin/tool1")
	assert.Equal(t, env["DEBUG_TOOL"], "plz-out/bin/tool1")
	assert.Equal(t, env["TMP_DIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["TMPDIR"], "/path/to/runtime/dir")
	assert.Equal(t, env["HOME"], "/path/to/runtime/dir")
	assert.Equal(t, env["TERM"], "my-term")
	assert.Equal(t, env["OUTS"], "out_file1")
	assert.Equal(t, env["OUT"], "out_file1")
	assert.Equal(t, env["TEST"], "out_file1")
}

func TestDeduplicateEnvVars(t *testing.T) {
	state := NewDefaultBuildState()
	state.NeedCoverage = true

	target := NewBuildTarget(NewBuildLabel("pkg", "t"))
	target.Test = new(TestFields)
	target.AddOutput("out_file1")
	target.AddDebugDatum(FileLabel{File: "data_file1", Package: "pkg"})
	target.Env = map[string]string{"COVERAGE": "wibble"}

	env := TestEnvironment(state, target, "/path/to/runtime/dir", 1)
	assert.Equal(t, env["COVERAGE"], "wibble")
}
