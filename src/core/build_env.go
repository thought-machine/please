package core

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/scm"
)

// A BuildEnv is a representation of the build environment that also knows how to log itself.
type BuildEnv map[string]string

// GeneralBuildEnvironment creates the shell env vars used for a command, not based
// on any specific target etc.
func GeneralBuildEnvironment(state *BuildState) BuildEnv {
	env := BuildEnv{
		// Need this for certain tools, for example sass
		"LANG": state.Config.Build.Lang,
		// Need to know these for certain rules.
		"ARCH": state.Arch.Arch,
		"OS":   state.Arch.OS,
		// These are slightly modified forms that are more convenient for some things.
		"XARCH": state.Arch.XArch(),
		"XOS":   state.Arch.XOS(),
	}

	if state.Config.Cpp.PkgConfigPath != "" {
		env["PKG_CONFIG_PATH"] = state.Config.Cpp.PkgConfigPath
	}

	env.Add(state.Config.GetBuildEnv())
	return env
}

// TargetEnvironment returns the basic parts of the build environment.
func TargetEnvironment(state *BuildState, target *BuildTarget) BuildEnv {
	env := GeneralBuildEnvironment(state)
	env["PKG"] = target.Label.PackageName
	env["PKG_DIR"] = target.PackageDir()
	env["NAME"] = target.Label.Name
	if state.Config.Remote.URL == "" || target.Local {
		// Expose the requested build config, but it is not available for remote execution.
		// TODO(peterebden): Investigate removing these env vars completely.
		env["BUILD_CONFIG"] = state.Config.Build.Config
		env["CONFIG"] = state.Config.Build.Config
	}
	if target.PassUnsafeEnv != nil {
		for _, e := range *target.PassUnsafeEnv {
			env[e] = os.Getenv(e)
		}
	}

	if target.PassEnv != nil {
		for _, e := range *target.PassEnv {
			env[e] = os.Getenv(e)
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

	env["TMP_DIR"] = tmpDir
	env["TMPDIR"] = tmpDir
	env["SRCS"] = strings.Join(sources, " ")
	env["OUTS"] = strings.Join(outEnv, " ")
	env["HOME"] = tmpDir
	// Set a consistent hash seed for Python. Important for build determinism.
	env["PYTHONHASHSEED"] = "42"

	// The OUT variable is only available on rules that have a single output.
	if len(outEnv) == 1 {
		env["OUT"] = resolveOut(outEnv[0], tmpDir, target.Sandbox)
	}
	// The SRC variable is only available on rules that have a single source file.
	if len(sources) == 1 {
		env["SRC"] = sources[0]
	}
	// Named source groups if the target declared any.
	for name, srcs := range target.NamedSources {
		paths := target.SourcePaths(state.Graph, srcs)
		// TODO(macripps): Quote these to prevent spaces from breaking everything (consider joining with NUL or sth?)
		env["SRCS_"+strings.ToUpper(name)] = strings.Join(paths, " ")
	}
	// Named output groups similarly.
	for name, outs := range target.DeclaredNamedOutputs() {
		outs = target.GetTmpOutputAll(outs)
		env["OUTS_"+strings.ToUpper(name)] = strings.Join(outs, " ")
	}
	// Tools
	env.Add(toolsEnv(state, target.AllTools(), target.namedTools, "", abs))
	// Secrets, again only if they declared any.
	if len(target.Secrets) > 0 {
		secrets := fs.ExpandHomePath(strings.Join(target.Secrets, ":"))
		secrets = strings.ReplaceAll(secrets, ":", " ")
		env["SECRETS"] = secrets
	}
	// NamedSecrets, if they declared any.
	for name, secrets := range target.NamedSecrets {
		secrets := fs.ExpandHomePath(strings.Join(secrets, ":"))
		env["SECRETS_"+strings.ToUpper(name)] = strings.ReplaceAll(secrets, ":", " ")
	}
	if target.Sandbox && len(state.Config.Sandbox.Dir) > 0 {
		env["SANDBOX_DIRS"] = strings.Join(state.Config.Sandbox.Dir, ",")
	}
	if state.Config.Bazel.Compatibility {
		// Obviously this is only a subset of the variables Bazel would expose, but there's
		// no point populating ones that we literally have no clue what they should be.
		// To be honest I don't terribly like these, I'm pretty sure that using $GENDIR in
		// your genrule is not a good sign.
		env["GENDIR"] = filepath.Join(RepoRoot, GenDir)
		env["BINDIR"] = filepath.Join(RepoRoot, BinDir)
	}

	return withUserProvidedEnv(target, env)
}

// userEnv adds the env variables passed to the build rule to the build env
// Sadly this can't be done as part of TargetEnv() target env as this requires the other
// env vars are set so they can be substituted.
func withUserProvidedEnv(target *BuildTarget, env BuildEnv) BuildEnv {
	for k, v := range target.Env {
		if strings.Contains(v, "$") {
			v = os.Expand(v, func(k string) string {
				if v, present := env[k]; present {
					return v
				}
				return "$" + k
			})
		}
		env[k] = v
	}
	return env
}

// TestEnvironment creates the environment variables for a test.
func TestEnvironment(state *BuildState, target *BuildTarget, testDir string, run int) BuildEnv {
	env := RuntimeEnvironment(state, target, filepath.IsAbs(testDir), true)
	resultsFile := filepath.Join(testDir, TestResultsFile)
	// Make this unintelligible to the consumer so it's at least hard for them to get cute with it.
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", target.Label, run)))

	env["TEST_DIR"] = testDir
	env["TMP_DIR"] = testDir
	env["TMPDIR"] = testDir
	env["HOME"] = testDir
	env["TEST_ARGS"] = strings.Join(state.TestArgs, ",")
	env["RESULTS_FILE"] = resultsFile
	// We shouldn't really have specific things like this here, but it really is just easier to set it.
	env["GTEST_OUTPUT"] = "xml:" + resultsFile
	env["PEX_NOCACHE"] = "true"
	env["_TEST_ID"] = base64.RawStdEncoding.EncodeToString(hash[:12])
	if state.NeedCoverage && !target.HasAnyLabel(state.Config.Test.DisableCoverage) {
		env["COVERAGE"] = "true"
		env["COVERAGE_FILE"] = filepath.Join(testDir, CoverageFile)
	}
	if len(target.Outputs()) > 0 {
		env["TEST"] = resolveOut(target.Outputs()[0], testDir, target.Test.Sandbox)
	}
	// Bit of a hack for gcov which needs access to its .gcno files.
	if target.HasLabel("cc") {
		env["GCNO_DIR"] = filepath.Join(RepoRoot, GenDir, target.Label.PackageName)
	}
	if state.DebugFailingTests {
		env["DEBUG_TEST_FAILURE"] = "true"
	}
	if target.Test.Sandbox && len(state.Config.Sandbox.Dir) > 0 {
		env["SANDBOX_DIRS"] = strings.Join(state.Config.Sandbox.Dir, ",")
	}
	if len(state.TestArgs) > 0 {
		env["TESTS"] = strings.Join(state.TestArgs, " ")
	}
	return withUserProvidedEnv(target, env)
}

// RunEnvironment creates the environment variables for a `plz run --env`.
func RunEnvironment(state *BuildState, target *BuildTarget, inTmpDir bool) BuildEnv {
	env := RuntimeEnvironment(state, target, true, inTmpDir)

	outEnv := target.Outputs()
	env["OUTS"] = strings.Join(outEnv, " ")
	// The OUT variable is only available on rules that have a single output.
	if len(outEnv) == 1 {
		env["OUT"] = resolveOut(outEnv[0], ".", false)
	}

	return withUserProvidedEnv(target, env)
}

// ExecEnvironment creates the environment variables for a `plz exec`.
func ExecEnvironment(state *BuildState, target *BuildTarget, execDir string) BuildEnv {
	env := RuntimeEnvironment(state, target, true, true)
	env["TMP_DIR"] = execDir
	env["TMPDIR"] = execDir
	env["HOME"] = execDir
	// This is used by programs that use display terminals for correct handling
	// of input and output in the terminal where the program is run.
	env["TERM"] = os.Getenv("TERM")

	outEnv := target.Outputs()
	// OUTS/OUT environment variables being always set is for backwards-compatibility.
	// Ideally, if the target is a test these variables shouldn't be set.
	env["OUTS"] = strings.Join(outEnv, " ")
	if len(outEnv) == 1 {
		env["OUT"] = resolveOut(outEnv[0], ".", target.Sandbox)
		if target.IsTest() {
			env["TEST"] = resolveOut(outEnv[0], ".", target.Test.Sandbox)
		}
	}

	return withUserProvidedEnv(target, env)
}

// RuntimeEnvironment is the base environment for runtime-based environments.
// Tools and data env variables are made available.
func RuntimeEnvironment(state *BuildState, target *BuildTarget, abs, inTmpDir bool) BuildEnv {
	env := TargetEnvironment(state, target)

	// Data
	env.Add(dataEnv(state, target.AllData(), target.NamedData, "", inTmpDir))

	if target.IsTest() {
		// Test tools
		env.Add(toolsEnv(state, target.AllTestTools(), target.NamedTestTools(), "", abs))
	}

	if target.Debug != nil {
		prefix := "DEBUG_"
		// Debug data
		env.Add(dataEnv(state, target.AllDebugData(), target.DebugNamedData(), prefix, inTmpDir))
		// Debug tools
		env.Add(toolsEnv(state, target.AllDebugTools(), target.Debug.namedTools, prefix, abs))
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
		prefix + "TOOLS": strings.Join(toolPaths(state, allTools, abs), " "),
	}
	if len(allTools) == 1 {
		env[prefix+"TOOL"] = toolPath(state, allTools[0], abs)
	}
	for name, tools := range namedTools {
		env[prefix+"TOOLS_"+strings.ToUpper(name)] = strings.Join(toolPaths(state, tools, abs), " ")
	}
	return env
}

// Creates data-related env variables
func dataEnv(state *BuildState, allData []BuildInput, namedData map[string][]BuildInput, prefix string, inTmpDir bool) BuildEnv {
	env := BuildEnv{
		prefix + "DATA": strings.Join(runtimeDataPaths(state.Graph, allData, !inTmpDir), " "),
	}
	for name, data := range namedData {
		env[prefix+"DATA_"+strings.ToUpper(name)] = strings.Join(runtimeDataPaths(state.Graph, data, !inTmpDir), " ")
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
		env.Add(stampEnv)
		env["STAMP_FILE"] = target.StampFileName()
		env["STAMP"] = encStamp
	}
	env["RULE_HASH"] = encStamp
	return env
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
		"SCM_COMMIT_DATE": commitDate,
		"SCM_REVISION":    revision,
		"SCM_DESCRIBE":    describe,
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
	if v, present := env[s]; present {
		return v
	}
	return ""
}

// Replace replaces the value of the given variable in this BuildEnv.
func (env BuildEnv) Replace(key, value string) {
	if _, present := env[key]; present {
		env[key] = value
	}
}

// Redacted implements the interface for our logging implementation.
func (env BuildEnv) Redacted() interface{} {
	r := make(BuildEnv, len(env))
	for k, v := range env {
		if strings.Contains(k, "SECRET") || strings.Contains(k, "PASSWORD") {
			v = "************"
		}
		r[k] = v
	}
	return r
}

// ToSlice converts this env into a list of env vars
func (env BuildEnv) ToSlice() []string {
	ret := make([]string, 0, len(env))
	for k, v := range env {
		ret = append(ret, k+"="+v)
	}
	sort.Strings(ret)
	return ret
}

// String implements the fmt.Stringer interface
func (env BuildEnv) String() string {
	return strings.Join(env.ToSlice(), "\n")
}

// Add adds the given set of environment variables to this one, overwriting on duplicates.
func (env BuildEnv) Add(that BuildEnv) {
	for k, v := range that {
		env[k] = v
	}
}
