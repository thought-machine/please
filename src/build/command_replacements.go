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
// $(out_location //path/to:target)
//   Expands to a path to the output of the given target, with the preceding plz-out/gen
//   or plz-out/bin etc. Useful when these things will be run by a user.
//
// In general it's a good idea to use these where possible in genrules rather than
// hardcoding specific paths.

package build

import (
	"encoding/base64"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"core"
)

var locationReplacement = regexp.MustCompile("\\$\\(location ([^\\)]+)\\)")
var locationsReplacement = regexp.MustCompile("\\$\\(locations ([^\\)]+)\\)")
var exeReplacement = regexp.MustCompile("\\$\\(exe ([^\\)]+)\\)")
var outExeReplacement = regexp.MustCompile("\\$\\(out_exe ([^\\)]+)\\)")
var outReplacement = regexp.MustCompile("\\$\\(out_location ([^\\)]+)\\)")
var dirReplacement = regexp.MustCompile("\\$\\(dir ([^\\)]+)\\)")
var hashReplacement = regexp.MustCompile("\\$\\(hash ([^\\)]+)\\)")

// Replace escape sequences in the target's command.
// For example, $(location :blah) -> the output of rule blah.
func replaceSequences(target *core.BuildTarget) string {
	return ReplaceSequences(target, target.GetCommand())
}

// Replace escape sequences in the given string.
func ReplaceSequences(target *core.BuildTarget, command string) string {
	return replaceSequencesInternal(target, command, false)
}

// Replace escape sequences in the given string when running a test.
func ReplaceTestSequences(target *core.BuildTarget, command string) string {
	if command == "" {
		// An empty test command implies running the test binary.
		return replaceSequencesInternal(target, fmt.Sprintf("$(exe :%s)", target.Label.Name), true)
	}
	return replaceSequencesInternal(target, command, true)
}

func replaceSequencesInternal(target *core.BuildTarget, command string, test bool) string {
	cmd := locationReplacement.ReplaceAllStringFunc(command, func(in string) string {
		return replaceSequence(target, in[11:len(in)-1], false, false, false, false, false, test)
	})
	cmd = locationsReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[12:len(in)-1], false, true, false, false, false, test)
	})
	cmd = exeReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[6:len(in)-1], true, false, false, false, false, test)
	})
	cmd = outReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[15:len(in)-1], false, false, false, true, false, test)
	})
	cmd = outExeReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[10:len(in)-1], true, false, false, true, false, test)
	})
	cmd = dirReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[6:len(in)-1], false, true, true, false, false, test)
	})
	cmd = hashReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		if !target.Stamp {
			panic(fmt.Sprintf("Target %s can't use $(hash ) replacements without stamp=True", target.Label))
		}
		return replaceSequence(target, in[7:len(in)-1], false, true, true, false, true, test)
	})
	if core.State.Config.Bazel.Compatibility {
		// Bazel allows several obscure Make-style variable expansions.
		// Our replacement here is not very principled but should work better than not doing it at all.
		cmd = strings.Replace(cmd, "$<", "$SRCS", -1)
		cmd = strings.Replace(cmd, "$(<)", "$SRCS", -1)
		cmd = strings.Replace(cmd, "$@D", "$TMP_DIR", -1)
		cmd = strings.Replace(cmd, "$(@D)", "$TMP_DIR", -1)
		cmd = strings.Replace(cmd, "$@", "$OUTS", -1)
		cmd = strings.Replace(cmd, "$(@)", "$OUTS", -1)
		// It also seemingly allows you to get away with this syntax, which means something
		// fairly different in Bash, but never mind.
		cmd = strings.Replace(cmd, "$(SRCS)", "$SRCS", -1)
		cmd = strings.Replace(cmd, "$(OUTS)", "$OUTS", -1)
	}
	// We would ideally check for this when doing matches above, but not easy in
	// Go since its regular expressions are actually regular and principled.
	return strings.Replace(cmd, "\\$", "$", -1)
}

// Replaces a single escape sequence in a command.
func replaceSequence(target *core.BuildTarget, in string, runnable, multiple, dir, outPrefix, hash, test bool) string {
	if core.LooksLikeABuildLabel(in) {
		label := core.ParseBuildLabel(in, target.Label.PackageName)
		return replaceSequenceLabel(target, label, in, runnable, multiple, dir, outPrefix, hash, test, true)
	}
	for _, src := range target.AllSources() {
		if label := src.Label(); label != nil && src.String() == in {
			return replaceSequenceLabel(target, *label, in, runnable, multiple, dir, outPrefix, hash, test, false)
		}
	}
	return quote(path.Join(target.Label.PackageName, in))
}

func replaceSequenceLabel(target *core.BuildTarget, label core.BuildLabel, in string, runnable, multiple, dir, outPrefix, hash, test, allOutputs bool) string {
	// Check this label is a dependency of the target, otherwise it's not allowed.
	if label == target.Label { // targets can always use themselves.
		return checkAndReplaceSequence(target, target, in, runnable, multiple, dir, outPrefix, hash, test, allOutputs, false)
	}
	deps := target.DependenciesFor(label)
	if len(deps) == 0 {
		panic(fmt.Sprintf("Rule %s can't use %s; doesn't depend on target %s", target.Label, in, label))
	}
	// TODO(pebers): this does not correctly handle the case where there are multiple deps here
	//               (but is better than the previous case where it never worked at all)
	return checkAndReplaceSequence(target, deps[0], in, runnable, multiple, dir, outPrefix, hash, test, allOutputs, target.IsTool(label))
}

func checkAndReplaceSequence(target, dep *core.BuildTarget, in string, runnable, multiple, dir, outPrefix, hash, test, allOutputs, tool bool) string {
	if allOutputs && !multiple && len(dep.Outputs()) != 1 {
		// Label must have only one output.
		panic(fmt.Sprintf("Rule %s can't use %s; %s has multiple outputs.", target.Label, in, dep.Label))
	} else if runnable && !dep.IsBinary {
		panic(fmt.Sprintf("Rule %s can't $(exe %s), it's not executable", target.Label, dep.Label))
	} else if runnable && len(dep.Outputs()) == 0 {
		panic(fmt.Sprintf("Rule %s is tagged as binary but produces no output.", dep.Label))
	}
	if hash {
		return base64.RawURLEncoding.EncodeToString(mustShortTargetHash(core.State, dep))
	}
	output := ""
	for _, out := range dep.Outputs() {
		if allOutputs || out == in {
			if tool {
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
	if runnable && dep.HasLabel("java_non_exe") {
		// The target is a Java target that isn't self-executable, hence it needs something to run it.
		output = "java -jar " + output
	}
	return strings.TrimRight(output, " ")
}

func fileDestination(target, dep *core.BuildTarget, out string, dir, outPrefix, test bool) string {
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
	return path.Join(outDir, output)
}
