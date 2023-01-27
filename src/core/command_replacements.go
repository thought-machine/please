// Replacement of sequences in genrule commands.
//
// Genrules can contain certain replacement variables which Please substitutes
// with locations of the actual thing before running.
// The following replacements are currently made:
//
// $(location //path/to:target)
//   Expands to the output of the given build rule. The rule can only have one
//   output (use $locations if there are multiple).
//
// $(locations //path/to:target)
//   Expands to all the outputs (space separated) of the given build rule.
//   Equivalent to $(location ...) for rules with a single output.
//
// $(exe //path/to:target)
//   Expands to a command to run the output of the given target from within a
//   genrule or test directory. For example,
//   java -jar path/to/target.jar.
//   The rule must be tagged as 'binary'.
//
// $(out_exe //path/to:target)
//   Expands to a command to run the output of the given target. For example,
//   java -jar plz-out/bin/path/to/target.jar.
//   The rule must be tagged as 'binary'.
//
// $(dir //path/to:target)
//   Expands to the package directory containing the outputs of the given target.
//   Useful for rules that have multiple outputs where you only need to know
//   what directory they're in.
//
// $(out_dir //path/to:target)
//   Expands to the out directory containing the outputs of the given target.
//   Useful for scripts that users that need to know the out directory fo a rule.
//
// $(out_location //path/to:target)
//   Expands to a path to the output of the given target, with the preceding plz-out/gen
//   or plz-out/bin etc. Useful when these things will be run by a user.
//
// $(out_locations //path/to:target)
//   Expands to a path(s) to the output of the given target, with the preceding plz-out/gen
//   or plz-out/bin etc. Useful when these things will be run by a user.
//
// $(worker //path/to:target)
//   Indicates that this target will be run by a remote worker process. The following
//   arguments are sent to the remote worker.
//   This is subject to some additional rules: it must appear initially in the command,
//   and if "&&" appears subsequently in the command, that part is run locally after
//   the worker has completed. All workers must be listed as tools of the rule.
//
// In general it's a good idea to use these where possible in genrules rather than
// hardcoding specific paths.

package core

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/peterebden/go-deferred-regex"

	"github.com/thought-machine/please/src/fs"
)

var locationReplacement = deferredregex.DeferredRegex{Re: `\$\(location ([^\)]+)\)`}
var locationsReplacement = deferredregex.DeferredRegex{Re: `\$\(locations ([^\)]+)\)`}
var exeReplacement = deferredregex.DeferredRegex{Re: `\$\(exe ([^\)]+)\)`}
var outExeReplacement = deferredregex.DeferredRegex{Re: `\$\(out_exe ([^\)]+)\)`}
var outReplacement = deferredregex.DeferredRegex{Re: `\$\(out_location ([^\)]+)\)`}
var outsReplacement = deferredregex.DeferredRegex{Re: `\$\(out_locations ([^\)]+)\)`}
var dirReplacement = deferredregex.DeferredRegex{Re: `\$\(dir ([^\)]+)\)`}
var outDirReplacement = deferredregex.DeferredRegex{Re: `\$\(out_dir ([^\)]+)\)`}
var hashReplacement = deferredregex.DeferredRegex{Re: `\$\(hash ([^\)]+)\)`}
var workerReplacement = deferredregex.DeferredRegex{Re: `^(.*)\$\(worker ([^\)]+)\) *([^&]*)(?: *&& *(.*))?$`}

// ReplaceSequences replaces escape sequences in the given string.
func ReplaceSequences(state *BuildState, target *BuildTarget, command string) (string, error) {
	return replaceSequencesInternal(state, target, command, false)
}

// ReplaceTestSequences replaces escape sequences in the given string when running a test.
func ReplaceTestSequences(state *BuildState, target *BuildTarget, command string) (string, error) {
	if command == "" {
		// An empty test command implies running the test binary.
		return replaceSequencesInternal(state, target, fmt.Sprintf("$(exe :%s)", target.Label.Name), true)
	} else if strings.HasPrefix(command, "$(worker") {
		_, _, cmd, err := workerAndArgs(state, target, command)
		return cmd, err
	}
	return replaceSequencesInternal(state, target, command, true)
}

// TestWorkerCommand returns the worker & its arguments (if any) for a test, and the command to run for the test itself.
func TestWorkerCommand(state *BuildState, target *BuildTarget) (string, string, string, error) {
	return workerAndArgs(state, target, target.GetTestCommand(state))
}

// WorkerCommandAndArgs returns the worker & its command (if any) and subsequent local command for the rule.
func WorkerCommandAndArgs(state *BuildState, target *BuildTarget) (string, string, string, error) {
	return workerAndArgs(state, target, target.GetCommand(state))
}

func workerAndArgs(state *BuildState, target *BuildTarget, command string) (string, string, string, error) {
	match := workerReplacement.FindStringSubmatch(command)
	if match == nil {
		cmd, err := ReplaceSequences(state, target, command)
		return "", "", cmd, err
	} else if match[1] != "" {
		panic("$(worker) replacements cannot have any commands preceding them.")
	}
	cmd1, err := replaceSequencesInternal(state, target, strings.TrimSpace(match[3]), false)
	if err != nil {
		return "", "", "", err
	}
	cmd2, err := replaceSequencesInternal(state, target, match[4], false)
	return replaceWorkerSequence(state, target, fs.ExpandHomePath(match[2]), true, false, false, true, false, false), cmd1, cmd2, err
}

func replaceSequencesInternal(state *BuildState, target *BuildTarget, command string, test bool) (cmd string, err error) {
	// TODO(peterebden): should probably just get rid of all the panics and thread errors around properly.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
			log.Debug(string(debug.Stack()))
		}
	}()
	cmd = locationReplacement.ReplaceAllStringFunc(command, func(in string) string {
		return replaceSequence(state, target, in[11:len(in)-1], false, false, false, false, false, test)
	})
	cmd = locationsReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[12:len(in)-1], false, true, false, false, false, test)
	})
	cmd = exeReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[6:len(in)-1], true, false, false, false, false, test)
	})
	cmd = outReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[15:len(in)-1], false, false, false, true, false, test)
	})
	cmd = outsReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[16:len(in)-1], false, true, false, true, false, test)
	})
	cmd = outExeReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[10:len(in)-1], true, false, false, true, false, test)
	})
	cmd = dirReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[6:len(in)-1], false, true, true, false, false, test)
	})
	cmd = outDirReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[10:len(in)-1], false, true, true, true, false, test)
	})
	cmd = hashReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(state, target, in[7:len(in)-1], false, true, true, false, true, test)
	})
	if state.Config.Bazel.Compatibility {
		// Bazel allows several obscure Make-style variable expansions.
		// Our replacement here is not very principled but should work better than not doing it at all.
		cmd = strings.ReplaceAll(cmd, "$<", "$SRCS")
		cmd = strings.ReplaceAll(cmd, "$(<)", "$SRCS")
		cmd = strings.ReplaceAll(cmd, "$@D", "$TMP_DIR")
		cmd = strings.ReplaceAll(cmd, "$(@D)", "$TMP_DIR")
		cmd = strings.ReplaceAll(cmd, "$@", "$OUTS")
		cmd = strings.ReplaceAll(cmd, "$(@)", "$OUTS")
		// It also seemingly allows you to get away with this syntax, which means something
		// fairly different in Bash, but never mind.
		cmd = strings.ReplaceAll(cmd, "$(SRCS)", "$SRCS")
		cmd = strings.ReplaceAll(cmd, "$(OUTS)", "$OUTS")
	}
	// We would ideally check for this when doing matches above, but not easy in
	// Go since its regular expressions are actually regular and principled.
	return strings.ReplaceAll(cmd, "\\$", "$"), nil
}

func splitEntryPoint(label string) (string, string) {
	if strings.Contains(label, "|") {
		parts := strings.Split(label, "|")
		return parts[0], parts[1]
	}
	return label, ""
}

// replaceSequence replaces a single escape sequence in a command.
func replaceSequence(state *BuildState, target *BuildTarget, in string, runnable, multiple, dir, outPrefix, hash, test bool) string {
	if LooksLikeABuildLabel(in) {
		in, ep := splitEntryPoint(in)
		label, err := TryParseBuildLabel(in, target.Label.PackageName, target.Label.Subrepo)
		if err != nil {
			panic(err)
		}
		return replaceSequenceLabel(state, target, label, ep, in, runnable, multiple, dir, outPrefix, hash, test, true)
	}
	for _, src := range sourcesOrTools(target, runnable) {
		if label, ok := src.Label(); ok && src.String() == in {
			return replaceSequenceLabel(state, target, label, "", in, runnable, multiple, dir, outPrefix, hash, test, false)
		} else if runnable && src.String() == in {
			return src.String()
		}
	}
	if hash {
		return base64.RawURLEncoding.EncodeToString(state.PathHasher.MustHash(filepath.Join(target.Label.PackageName, in), target.HashLastModified()))
	}
	if strings.HasPrefix(in, "/") {
		return in // Absolute path, probably on a tool or system src.
	}
	return quote(filepath.Join(target.Label.PackageName, in))
}

// replaceWorkerSequence is like replaceSequence but for worker commands, which do not
// prefix the target's directory if it's not a build label.
func replaceWorkerSequence(state *BuildState, target *BuildTarget, in string, runnable, multiple, dir, outPrefix, hash, test bool) string {
	if !LooksLikeABuildLabel(in) {
		return in
	}
	return replaceSequence(state, target, in, runnable, multiple, dir, outPrefix, hash, test)
}

// sourcesOrTools returns either the tools of a target if runnable is true, otherwise its sources.
func sourcesOrTools(target *BuildTarget, runnable bool) []BuildInput {
	if runnable {
		return target.Tools
	}
	return target.AllSources()
}

func replaceSequenceLabel(state *BuildState, target *BuildTarget, label BuildLabel, ep string, in string, runnable, multiple, dir, outPrefix, hash, test, allOutputs bool) string {
	// Check this label is a dependency of the target, otherwise it's not allowed.
	if label == target.Label { // targets can always use themselves.
		return checkAndReplaceSequence(state, target, target, ep, in, runnable, multiple, dir, outPrefix, hash, test, allOutputs, false)
	}
	// TODO(jpoole): This doesn't handle tools when cross compiling. ///freebsd_amd64//tools:tool
	// will not match the tool //tools:tool
	deps := target.DependenciesFor(label)
	if len(deps) == 0 {
		panic(fmt.Sprintf("Rule %s can't use %s; doesn't depend on target %s", target.Label, in, label))
	}
	// TODO(pebers): this does not correctly handle the case where there are multiple deps here
	//               (but is better than the previous case where it never worked at all)
	return checkAndReplaceSequence(state, target, deps[0], ep, in, runnable, multiple, dir, outPrefix, hash, test, allOutputs, target.IsTool(label))
}

func checkAndReplaceSequence(state *BuildState, target, dep *BuildTarget, ep, in string, runnable, multiple, dir, outPrefix, hash, test, allOutputs, tool bool) string {
	if allOutputs && !multiple && len(dep.Outputs()) > 1 && ep == "" {
		// Label must have only one output.
		panic(fmt.Sprintf("Rule %s can't use %s; %s has multiple outputs.", target.Label, in, dep.Label))
	} else if runnable && !dep.IsBinary {
		panic(fmt.Sprintf("Rule %s can't $(exe %s), it's not executable", target.Label, dep.Label))
	} else if runnable && len(dep.Outputs()) == 0 {
		panic(fmt.Sprintf("Rule %s is tagged as binary but produces no output.", dep.Label))
	} else if test && tool {
		panic(fmt.Sprintf("Rule %s uses %s in its test command, but tools are not accessible at test time", target, dep))
	}
	if hash {
		h, err := state.TargetHasher.OutputHash(dep)
		if err != nil {
			panic(err)
		}
		return base64.RawURLEncoding.EncodeToString(h)
	}
	output := ""
	if ep == "" {
		for _, out := range dep.Outputs() {
			if allOutputs || out == in {
				if tool && !state.WillRunRemotely(target) {
					abs, err := filepath.Abs(handleDir(dep.OutDir(), out, dir))
					if err != nil {
						log.Fatalf("Couldn't calculate relative path: %s", err)
					}
					output += quote(abs) + " "
				} else {
					output += quote(fileDestination(target, dep, out, dir, outPrefix, test)) + " "
				}
				if dir {
					break
				}
			}
		}
	} else {
		out, ok := dep.EntryPoints[ep]
		if !ok {
			log.Fatalf("%v has no entry point %s", dep, ep)
		}
		output = quote(fileDestination(target, dep, out, dir, outPrefix, test))
	}

	return strings.TrimRight(output, " ")
}

func fileDestination(target, dep *BuildTarget, out string, dir, outPrefix, test bool) string {
	if outPrefix {
		return handleDir(dep.OutDir(), out, dir)
	}
	if test && target == dep {
		// Slightly fiddly case because tests put binaries in a possibly slightly unusual place.
		return "./" + out
	}
	return handleDir(dep.Label.PackageName, out, dir)
}

// Encloses the given string in quotes if needed.
func quote(s string) string {
	if strings.ContainsAny(s, "|&;()<>") {
		return "\"" + s + "\""
	}
	return s
}

// handleDir chooses either the out dir or the actual output location depending on the 'dir' flag.
func handleDir(outDir, output string, dir bool) string {
	if dir {
		return outDir
	}
	return filepath.Join(outDir, output)
}
