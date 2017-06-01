package core

import (
	"encoding/base64"
	"os"
	"path"
	"regexp"
	"runtime"
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

// GeneralBuildEnvironment creates the shell env vars used for a command, not based
// on any specific target etc.
func GeneralBuildEnvironment(config *Configuration) []string {
	env := []string{
		// Need to know these for certain rules, particularly Go rules.
		"ARCH=" + runtime.GOARCH,
		"OS=" + runtime.GOOS,
		// Need this for certain tools, for example sass
		"LANG=" + config.Please.Lang,
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
	return env
}

// BuildEnvironment creates the shell env vars to be passed
// into the exec.Command calls made by plz. Use test=true for plz test targets.
func BuildEnvironment(state *BuildState, target *BuildTarget, test bool) []string {
	sources := target.AllSourcePaths(state.Graph)
	env := GeneralBuildEnvironment(state.Config)
	env = append(env, "PKG="+target.Label.PackageName, "PKG_DIR="+target.Label.PackageDir())
	if !test {
		env = append(env,
			"TMP_DIR="+path.Join(RepoRoot, target.TmpDir()),
			"SRCS="+strings.Join(sources, " "),
			"OUTS="+strings.Join(target.Outputs(), " "),
			"NAME="+target.Label.Name,
		)
		env = append(env, "TOOLS="+strings.Join(toolPaths(state, target.Tools), " "))
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
			env = append(env, "SECRETS="+strings.Join(target.Secrets, " "))
		}
		if state.Config.Bazel.Compatibility {
			// Obviously this is only a subset of the variables Bazel would expose, but there's
			// no point populating ones that we literally have no clue what they should be.
			// To be honest I don't terribly like these, I'm pretty sure that using $GENDIR in
			// your genrule is not a good sign.
			env = append(env, "GENDIR="+path.Join(RepoRoot, GenDir))
			env = append(env, "BINDIR="+path.Join(RepoRoot, BinDir))
		}
	} else {
		env = append(env, "TEST_DIR="+path.Join(RepoRoot, target.TestDir()))
		env = append(env, "TEST_ARGS="+strings.Join(state.TestArgs, ","))
		if state.NeedCoverage {
			env = append(env, "COVERAGE=true", "COVERAGE_FILE="+path.Join(RepoRoot, target.TestDir(), "test.coverage"))
		}
		if len(target.Outputs()) > 0 {
			env = append(env, "TEST="+path.Join(RepoRoot, target.TestDir(), target.Outputs()[0]))
		}
		// Bit of a hack for gcov which needs access to its .gcno files.
		if target.HasLabel("cc") {
			env = append(env, "GCNO_DIR="+path.Join(RepoRoot, GenDir, target.Label.PackageName))
		}
	}
	return env
}

// StampedBuildEnvironment returns the shell env vars to be passed into exec.Command.
// Optionally includes a stamp if the target is marked as such.
func StampedBuildEnvironment(state *BuildState, target *BuildTarget, test bool, stamp []byte) []string {
	env := BuildEnvironment(state, target, test)
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

// ReplaceEnvironment returns a function suitable for passing to os.Expand to replace environment
// variables from an earlier call to BuildEnvironment.
func ReplaceEnvironment(env []string) func(string) string {
	return func(s string) string {
		for _, e := range env {
			if strings.HasPrefix(e, s+"=") {
				return e[len(s)+1:]
			}
		}
		return ""
	}
}
