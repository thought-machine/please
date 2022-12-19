package core

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/scm"
)

// A BuildEnv is a representation of the build environment that also knows how to log itself.
type BuildEnv []string

// GeneralBuildEnvironment creates the shell env vars used for a command, not based
// on any specific target etc.
func GeneralBuildEnvironment(state *BuildState) BuildEnv {
	env := BuildEnv{
		// Need this for certain tools, for example sass
		"LANG=" + state.Config.Build.Lang,
		// Need to know these for certain rules.
		"ARCH=" + state.Arch.Arch,
		"OS=" + state.Arch.OS,
		// These are slightly modified forms that are more convenient for some things.
		"XARCH=" + state.Arch.XArch(),
		"XOS=" + state.Arch.XOS(),
	}

	if !state.Config.FeatureFlags.ExcludeGoRules {
		env = append(env, "GOARCH="+state.Arch.GoArch(), "GOOS="+state.Arch.OS)
	}

	if state.Config.Cpp.PkgConfigPath != "" {
		env = append(env, "PKG_CONFIG_PATH="+state.Config.Cpp.PkgConfigPath)
	}

	return append(env, state.Config.GetBuildEnv()...)
}

// TargetEnvironment returns the basic parts of the build environment.
func TargetEnvironment(state *BuildState, target *BuildTarget) BuildEnv {
	env := append(GeneralBuildEnvironment(state),
		"PKG="+target.Label.PackageName,
		"PKG_DIR="+target.PackageDir(),
		"NAME="+target.Label.Name,
	)
	if state.Config.Remote.URL == "" || target.Local {
		// Expose the requested build config, but it is not available for remote execution.
		// TODO(peterebden): Investigate removing these env vars completely.
		env = append(env, "BUILD_CONFIG="+state.Config.Build.Config, "CONFIG="+state.Config.Build.Config)
	}
	if target.PassUnsafeEnv != nil {
		for _, e := range *target.PassUnsafeEnv {
			env = append(env, e+"="+os.Getenv(e))
		}
	}

	if target.PassEnv != nil {
		for _, e := range *target.PassEnv {
			env = append(env, e+"="+os.Getenv(e))
		}
	}
	return env
}

// BuildEnvironment creates the shell env vars to be passed into the exec.Command calls made by plz.
// Note that we lie about the location of HOME in order to keep some tools happy.
// We read this as being slightly more POSIX-compliant than not having it set at all...
func BuildEnvironment(state *BuildState, target *BuildTarget, tmpDir string) BuildEnv {
	env := TargetEnvironment(state, target)
	sources := target.AllSourcePaths(state.Graph)
	outEnv := target.GetTmpOutputAll(target.Outputs())
	abs := filepath.IsAbs(tmpDir)

	env = append(env,
		"TMP_DIR="+tmpDir,
		"TMPDIR="+tmpDir,
		"SRCS="+strings.Join(sources, " "),
		"OUTS="+strings.Join(outEnv, " "),
		"HOME="+tmpDir,
		// Set a consistent hash seed for Python. Important for build determinism.
		"PYTHONHASHSEED=42",
	)
	// The OUT variable is only available on rules that have a single output.
	if len(outEnv) == 1 {
		env = append(env, "OUT="+resolveOut(outEnv[0], tmpDir, target.Sandbox))
	}
	// The SRC variable is only available on rules that have a single source file.
	if len(sources) == 1 {
		env = append(env, "SRC="+sources[0])
	}
	// Named source groups if the target declared any.
	for name, srcs := range target.NamedSources {
		paths := target.SourcePaths(state.Graph, srcs)
		// TODO(macripps): Quote these to prevent spaces from breaking everything (consider joining with NUL or sth?)
		env = append(env, "SRCS_"+strings.ToUpper(name)+"="+strings.Join(paths, " "))
	}
	// Named output groups similarly.
	for name, outs := range target.DeclaredNamedOutputs() {
		outs = target.GetTmpOutputAll(outs)
		env = append(env, "OUTS_"+strings.ToUpper(name)+"="+strings.Join(outs, " "))
	}
	// Tools
	env = append(env, toolsEnv(state, target.AllTools(), target.namedTools, "", abs)...)
	// Secrets, again only if they declared any.
	if len(target.Secrets) > 0 {
		secrets := "SECRETS=" + fs.ExpandHomePath(strings.Join(target.Secrets, ":"))
		secrets = strings.ReplaceAll(secrets, ":", " ")
		env = append(env, secrets)
	}
	// NamedSecrets, if they declared any.
	for name, secrets := range target.NamedSecrets {
		secrets := "SECRETS_" + strings.ToUpper(name) + "=" + fs.ExpandHomePath(strings.Join(secrets, ":"))
		secrets = strings.ReplaceAll(secrets, ":", " ")
		env = append(env, secrets)
	}
	if target.Sandbox && len(state.Config.Sandbox.Dir) > 0 {
		env = append(env, "SANDBOX_DIRS="+strings.Join(state.Config.Sandbox.Dir, ","))
	}
	if state.Config.Bazel.Compatibility {
		// Obviously this is only a subset of the variables Bazel would expose, but there's
		// no point populating ones that we literally have no clue what they should be.
		// To be honest I don't terribly like these, I'm pretty sure that using $GENDIR in
		// your genrule is not a good sign.
		env = append(env, "GENDIR="+filepath.Join(RepoRoot, GenDir))
		env = append(env, "BINDIR="+filepath.Join(RepoRoot, BinDir))
	}

	return withUserProvidedEnv(target, env)
}

// userEnv adds the env variables passed to the build rule to the build env
// Sadly this can't be done as part of TargetEnv() target env as this requires the other
// env vars are set so they can be substituted.
func withUserProvidedEnv(target *BuildTarget, env BuildEnv) BuildEnv {
	for k, v := range target.Env {
		for _, kv := range env {
			i := strings.Index(kv, "=")
			key, value := kv[:i], kv[(i+1):]
			v = strings.ReplaceAll(v, "$"+key, value)
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// TestEnvironment creates the environment variables for a test.
func TestEnvironment(state *BuildState, target *BuildTarget, testDir string) BuildEnv {
	env := RuntimeEnvironment(state, target, filepath.IsAbs(testDir), true)
	resultsFile := filepath.Join(testDir, TestResultsFile)

	env = append(env,
		"TEST_DIR="+testDir,
		"TMP_DIR="+testDir,
		"TMPDIR="+testDir,
		"HOME="+testDir,
		"TEST_ARGS="+strings.Join(state.TestArgs, ","),
		"RESULTS_FILE="+resultsFile,
		// We shouldn't really have specific things like this here, but it really is just easier to set it.
		"GTEST_OUTPUT=xml:"+resultsFile,
		"PEX_NOCACHE=true",
	)
	if state.NeedCoverage && !target.HasAnyLabel(state.Config.Test.DisableCoverage) {
		env = append(env,
			"COVERAGE=true",
			"COVERAGE_FILE="+filepath.Join(testDir, CoverageFile),
		)
	}
	if len(target.Outputs()) > 0 {
		env = append(env, "TEST="+resolveOut(target.Outputs()[0], testDir, target.Test.Sandbox))
	}
	// Bit of a hack for gcov which needs access to its .gcno files.
	if target.HasLabel("cc") {
		env = append(env, "GCNO_DIR="+filepath.Join(RepoRoot, GenDir, target.Label.PackageName))
	}
	if state.DebugFailingTests {
		env = append(env, "DEBUG_TEST_FAILURE=true")
	}
	if target.Test.Sandbox && len(state.Config.Sandbox.Dir) > 0 {
		env = append(env, "SANDBOX_DIRS="+strings.Join(state.Config.Sandbox.Dir, ","))
	}
	if len(state.TestArgs) > 0 {
		env = append(env, "TESTS="+strings.Join(state.TestArgs, " "))
	}
	return withUserProvidedEnv(target, env)
}

// RunEnvironment creates the environment variables for a `plz run --env`.
func RunEnvironment(state *BuildState, target *BuildTarget, inTmpDir bool) BuildEnv {
	env := RuntimeEnvironment(state, target, true, inTmpDir)

	outEnv := target.Outputs()
	env = append(env, "OUTS="+strings.Join(outEnv, " "))
	// The OUT variable is only available on rules that have a single output.
	if len(outEnv) == 1 {
		env = append(env, "OUT="+resolveOut(outEnv[0], ".", false))
	}

	return withUserProvidedEnv(target, env)
}

// ExecEnvironment creates the environment variables for a `plz exec`.
func ExecEnvironment(state *BuildState, target *BuildTarget, execDir string) BuildEnv {
	env := append(RuntimeEnvironment(state, target, true, true),
		"TMP_DIR="+execDir,
		"TMPDIR="+execDir,
		"HOME="+execDir,
		// This is used by programs that use display terminals for correct handling
		// of input and output in the terminal where the program is run.
		"TERM="+os.Getenv("TERM"),
	)

	outEnv := target.Outputs()
	// OUTS/OUT environment variables being always set is for backwards-compatibility.
	// Ideally, if the target is a test these variables shouldn't be set.
	env = append(env, "OUTS="+strings.Join(outEnv, " "))
	if len(outEnv) == 1 {
		env = append(env, "OUT="+resolveOut(outEnv[0], ".", target.Sandbox))
		if target.IsTest() {
			env = append(env, "TEST="+resolveOut(outEnv[0], ".", target.Test.Sandbox))
		}
	}

	return withUserProvidedEnv(target, env)
}

// RuntimeEnvironment is the base environment for runtime-based environments.
// Tools and data env variables are made available.
func RuntimeEnvironment(state *BuildState, target *BuildTarget, abs, inTmpDir bool) BuildEnv {
	env := TargetEnvironment(state, target)

	// Data
	env = append(env, dataEnv(state, target.AllData(), target.NamedData, "", inTmpDir)...)

	if target.IsTest() {
		// Test tools
		env = append(env, toolsEnv(state, target.AllTestTools(), target.NamedTestTools(), "", abs)...)
	}

	if target.Debug != nil {
		prefix := "DEBUG_"
		// Debug data
		env = append(env, dataEnv(state, target.AllDebugData(), target.DebugNamedData(), prefix, inTmpDir)...)
		// Debug tools
		env = append(env, toolsEnv(state, target.AllDebugTools(), target.Debug.namedTools, prefix, abs)...)
	}

	return env
}

// Handles resolution of OUT files
func resolveOut(out string, dir string, sandbox bool) string {
	// Bit of a hack; ideally we would be unaware of the sandbox here.
	if sandbox && runtime.GOOS == "linux" && !strings.HasPrefix(RepoRoot, "/tmp/") && dir != "." {
		return filepath.Join(SandboxDir, out)
	}
	return filepath.Join(dir, out)
}

// Creates tool-related env variables
func toolsEnv(state *BuildState, allTools []BuildInput, namedTools map[string][]BuildInput, prefix string, abs bool) BuildEnv {
	env := BuildEnv{
		prefix + "TOOLS=" + strings.Join(toolPaths(state, allTools, abs), " "),
	}
	if len(allTools) == 1 {
		env = append(env, prefix+"TOOL="+toolPath(state, allTools[0], abs))
	}
	for name, tools := range namedTools {
		env = append(env, prefix+"TOOLS_"+strings.ToUpper(name)+"="+strings.Join(toolPaths(state, tools, abs), " "))
	}
	return env
}

// Creates data-related env variables
func dataEnv(state *BuildState, allData []BuildInput, namedData map[string][]BuildInput, prefix string, inTmpDir bool) BuildEnv {
	env := BuildEnv{
		prefix + "DATA=" + strings.Join(runtimeDataPaths(state.Graph, allData, !inTmpDir), " "),
	}
	for name, data := range namedData {
		env = append(env, prefix+"DATA_"+strings.ToUpper(name)+"="+strings.Join(runtimeDataPaths(state.Graph, data, !inTmpDir), " "))
	}
	return env
}

func runtimeDataPaths(graph *BuildGraph, data []BuildInput, fullPath bool) []string {
	paths := make([]string, 0, len(data))
	for _, in := range data {
		if fullPath {
			paths = append(paths, in.FullPaths(graph)...)
		} else {
			paths = append(paths, in.Paths(graph)...)
		}
	}
	return paths
}

// StampedBuildEnvironment returns the shell env vars to be passed into exec.Command.
// Optionally includes a stamp if asked.
func StampedBuildEnvironment(state *BuildState, target *BuildTarget, stamp []byte, tmpDir string, shouldStamp bool) BuildEnv {
	env := BuildEnvironment(state, target, tmpDir)
	encStamp := base64.RawURLEncoding.EncodeToString(stamp)
	if shouldStamp {
		stampEnvOnce.Do(initStampEnv)
		env = append(env, stampEnv...)
		env = append(env, "STAMP_FILE="+target.StampFileName())
		env = append(env, "STAMP="+encStamp)
	}
	return append(env, "RULE_HASH="+encStamp)
}

// stampEnv is the generic (i.e. non-target-specific) environment variables we pass to a
// build rule marked with stamp=True.
var stampEnv BuildEnv
var stampEnvOnce sync.Once

func initStampEnv() {
	repoScm := scm.NewFallback(RepoRoot)
	var wg sync.WaitGroup
	var revision, commitDate, describe string
	wg.Add(2)
	go func() {
		revision = repoScm.CurrentRevIdentifier(true)
		describe = repoScm.DescribeIdentifier(revision)
		wg.Done()
	}()
	go func() {
		commitDate = repoScm.CurrentRevDate("20060102")
		wg.Done()
	}()
	wg.Wait()
	stampEnv = BuildEnv{
		"SCM_COMMIT_DATE=" + commitDate,
		"SCM_REVISION=" + revision,
		"SCM_DESCRIBE=" + describe,
	}
}

func toolPath(state *BuildState, tool BuildInput, abs bool) string {
	if label, ok := tool.Label(); ok {
		entryPoint := ""
		if o, ok := tool.(AnnotatedOutputLabel); ok {
			entryPoint = o.Annotation
		}
		path := state.Graph.TargetOrDie(label).toolPath(abs, entryPoint)
		if !strings.Contains(path, "/") {
			path = "./" + path
		}
		return path
	} else if abs {
		return tool.Paths(state.Graph)[0]
	}
	return tool.LocalPaths(state.Graph)[0]
}

func toolPaths(state *BuildState, tools []BuildInput, abs bool) []string {
	ret := make([]string, len(tools))
	for i, tool := range tools {
		ret[i] = toolPath(state, tool, abs)
	}
	return ret
}

// ReplaceEnvironment is a function suitable for passing to os.Expand to replace environment
// variables from this BuildEnv.
func (env BuildEnv) ReplaceEnvironment(s string) string {
	for _, e := range env {
		if strings.HasPrefix(e, s+"=") {
			return e[len(s)+1:]
		}
	}
	return ""
}

// Replace replaces the value of the given variable in this BuildEnv.
func (env BuildEnv) Replace(key, value string) {
	key += "="
	for i, e := range env {
		if strings.HasPrefix(e, key) {
			env[i] = key + value
		}
	}
}

// Redacted implements the interface for our logging implementation.
func (env BuildEnv) Redacted() interface{} {
	r := make(BuildEnv, len(env))
	for i, e := range env {
		r[i] = e
		split := strings.SplitN(e, "=", 2)
		if len(split) == 2 && (strings.Contains(split[0], "SECRET") || strings.Contains(split[0], "PASSWORD")) {
			r[i] = split[0] + "=" + "************"
		}
	}
	return r
}

// String implements the fmt.Stringer interface
func (env BuildEnv) String() string {
	return strings.Join(env, "\n")
}
