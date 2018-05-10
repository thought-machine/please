package core

import (
	"encoding/base64"
	"os"
	"path"
	"regexp"
	"strings"
)

var home = os.Getenv("HOME")
var homeRex = regexp.MustCompile("(?:^|:)(~(?:[/:]|$))")

// ExpandHomePath expands all prefixes of ~ without a user specifier TO $HOME.
func ExpandHomePath(path string) string {
	return homeRex.ReplaceAllStringFunc(path, func(subpath string) string {
		return strings.Replace(subpath, "~", home, -1)
	})
}

// A BuildEnv is a representation of the build environment that also knows how to log itself.
type BuildEnv []string

// GeneralBuildEnvironment creates the shell env vars used for a command, not based
// on any specific target etc.
func GeneralBuildEnvironment(config *Configuration) BuildEnv {
	env := BuildEnv{
		// Need this for certain tools, for example sass
		"LANG=" + config.Build.Lang,
		// Use a restricted PATH; it'd be easier for the user if we pass it through
		// but really external environment variables shouldn't affect this.
		// The only concession is that ~ is expanded as the user's home directory
		// in PATH entries.
		"PATH=" + ExpandHomePath(strings.Join(config.Build.Path, ":")),
		// Expose the requested build config. We might also want to expose
		// the command that's actually running (although typically this is more useful,
		// because targets using this want to avoid defining different commands).
		"BUILD_CONFIG=" + config.Build.Config,
	}
	if config.Go.GoRoot != "" {
		env = append(env, "GOROOT="+config.Go.GoRoot)
	}
	if config.Cpp.PkgConfigPath != "" {
		env = append(env, "PKG_CONFIG_PATH="+config.Cpp.PkgConfigPath)
	}
	return append(env, config.GetBuildEnv()...)
}

// buildEnvironment returns the basic parts of the build environment.
func buildEnvironment(state *BuildState, target *BuildTarget) BuildEnv {
	return append(GeneralBuildEnvironment(state.Config),
		"PKG="+target.Label.PackageName,
		"PKG_DIR="+target.Subrepo.MakeRelativeName(target.Label.PackageDir()),
		"NAME="+target.Label.Name,
		"CONFIG="+state.Config.Build.Config,
	)
}

// BuildEnvironment creates the shell env vars to be passed into the exec.Command calls made by plz.
// Note that we lie about the location of HOME in order to keep some tools happy.
// We read this as being slightly more POSIX-compliant than not having it set at all...
func BuildEnvironment(state *BuildState, target *BuildTarget) BuildEnv {
	env := buildEnvironment(state, target)
	sources := target.AllSourcePaths(state.Graph)
	tmpDir := path.Join(RepoRoot, target.TmpDir())
	env = append(env,
		"TMP_DIR="+tmpDir,
		"TMPDIR="+tmpDir,
		"SRCS="+strings.Join(sources, " "),
		"OUTS="+strings.Join(target.Outputs(), " "),
		"HOME="+tmpDir,
		"TOOLS="+strings.Join(toolPaths(state, target.Tools), " "),
		// Set a consistent hash seed for Python. Important for build determinism.
		"PYTHONHASHSEED=42",
	)
	// The OUT variable is only available on rules that have a single output.
	if len(target.Outputs()) == 1 {
		env = append(env, "OUT="+path.Join(RepoRoot, target.TmpDir(), target.Outputs()[0]))
	}
	// The SRC variable is only available on rules that have a single source file.
	if len(sources) == 1 {
		env = append(env, "SRC="+sources[0])
	}
	// Similarly, TOOL is only available on rules with a single tool.
	if len(target.Tools) == 1 {
		env = append(env, "TOOL="+toolPath(state, target.Tools[0]))
	}
	// Named source groups if the target declared any.
	for name, srcs := range target.NamedSources {
		paths := target.SourcePaths(state.Graph, srcs)
		env = append(env, "SRCS_"+strings.ToUpper(name)+"="+strings.Join(paths, " "))
	}
	// Named output groups similarly.
	for name, outs := range target.DeclaredNamedOutputs() {
		env = append(env, "OUTS_"+strings.ToUpper(name)+"="+strings.Join(outs, " "))
	}
	// Named tools as well.
	for name, tools := range target.namedTools {
		env = append(env, "TOOLS_"+strings.ToUpper(name)+"="+strings.Join(toolPaths(state, tools), " "))
	}
	// Secrets, again only if they declared any.
	if len(target.Secrets) > 0 {
		env = append(env, "SECRETS="+ExpandHomePath(strings.Join(target.Secrets, " ")))
	}
	if state.Config.Bazel.Compatibility {
		// Obviously this is only a subset of the variables Bazel would expose, but there's
		// no point populating ones that we literally have no clue what they should be.
		// To be honest I don't terribly like these, I'm pretty sure that using $GENDIR in
		// your genrule is not a good sign.
		env = append(env, "GENDIR="+path.Join(RepoRoot, GenDir))
		env = append(env, "BINDIR="+path.Join(RepoRoot, BinDir))
	}
	return env
}

// TestEnvironment creates the environment variables for a test.
func TestEnvironment(state *BuildState, target *BuildTarget, testDir string) BuildEnv {
	env := buildEnvironment(state, target)
	resultsFile := path.Join(testDir, "test.results")
	env = append(env,
		"TEST_DIR="+testDir,
		"TMP_DIR="+testDir,
		"TMPDIR="+testDir,
		"TEST_ARGS="+strings.Join(state.TestArgs, ","),
		"RESULTS_FILE="+resultsFile,
		// We shouldn't really have specific things like this here, but it really is just easier to set it.
		"GTEST_OUTPUT=xml:"+resultsFile,
	)
	// Ideally we would set this to something useful even within a container, but it ends
	// up being /tmp/test or something which just confuses matters.
	if !target.Containerise {
		env = append(env, "HOME="+testDir)
	}
	if state.NeedCoverage {
		env = append(env, "COVERAGE=true", "COVERAGE_FILE="+path.Join(testDir, "test.coverage"))
	}
	if len(target.Outputs()) > 0 {
		env = append(env, "TEST="+path.Join(testDir, target.Outputs()[0]))
	}
	if len(target.Data) > 0 {
		env = append(env, "DATA="+strings.Join(target.AllData(state.Graph), " "))
	}
	// Bit of a hack for gcov which needs access to its .gcno files.
	if target.HasLabel("cc") {
		env = append(env, "GCNO_DIR="+path.Join(RepoRoot, GenDir, target.Label.PackageName))
	}
	if state.DebugTests {
		env = append(env, "DEBUG=true")
	}
	return env
}

// StampedBuildEnvironment returns the shell env vars to be passed into exec.Command.
// Optionally includes a stamp if the target is marked as such.
func StampedBuildEnvironment(state *BuildState, target *BuildTarget, stamp []byte) BuildEnv {
	env := BuildEnvironment(state, target)
	if target.Stamp {
		return append(env, "STAMP="+base64.RawURLEncoding.EncodeToString(stamp))
	}
	return env
}

func toolPath(state *BuildState, tool BuildInput) string {
	label := tool.Label()
	if label != nil {
		return state.Graph.TargetOrDie(*label).toolPath()
	}
	return tool.Paths(state.Graph)[0]
}

func toolPaths(state *BuildState, tools []BuildInput) []string {
	ret := make([]string, len(tools))
	for i, tool := range tools {
		ret[i] = toolPath(state, tool)
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
	key = key + "="
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
