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
// $(location_pairs //path/to:target)
//   Expands to pairs of absolute and relative paths for all the outputs of
//   the given build rule. Relative paths are relative to their package, so a
//   rule in package //d/e with outputs = ['a.txt', 'b/c.txt'] would receive
//   '/plz-out/gen/d/e/a.txt a.txt /plz-out/gen/d/e/b/c.txt b/c.txt' from this substitution.
//   This is a fairly specific one used mostly by filegroups.
//
// In general it's a good idea to use these where possible in genrules rather than
// hardcoding specific paths.

package build

import (
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
var outReplacement = regexp.MustCompile("\\$\\(out_location ([^\\)]+)\\)")
var dirReplacement = regexp.MustCompile("\\$\\(dir ([^\\)]+)\\)")
var locationPairsReplacement = regexp.MustCompile("\\$\\(location_pairs ([^\\)]+)\\)")

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
		return replaceSequence(target, in[15:len(in)-1], false, false, false, false, true, test)
	})
	cmd = dirReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[6:len(in)-1], false, true, false, true, false, test)
	})
	cmd = locationPairsReplacement.ReplaceAllStringFunc(cmd, func(in string) string {
		return replaceSequence(target, in[17:len(in)-1], false, true, true, false, false, test)
	})
	// TODO(pebers): We should check for this when doing matches above, but not easy in
	//               Go since its regular expressions are actually regular and principled.
	return strings.Replace(cmd, "\\$", "$", -1)
}

// Replaces a single escape sequence in a command.
func replaceSequence(target *core.BuildTarget, in string, runnable, multiple, pairs, dir, outPrefix, test bool) string {
	if core.LooksLikeABuildLabel(in) {
		label, file := core.ParseBuildFileLabel(in, target.Label.PackageName)
		return replaceSequenceLabel(target, label, file, in, runnable, multiple, pairs, dir, outPrefix, test, true)
	}
	for src := range target.AllSources() {
		if label := src.Label(); label != nil && src.String() == in {
			return replaceSequenceLabel(target, *label, "", in, runnable, multiple, pairs, dir, outPrefix, test, false)
		}
	}
	if pairs {
		return quote(path.Join(core.RepoRoot, target.Label.PackageName, in)) + " " + quote(in)
	} else {
		return quote(path.Join(target.Label.PackageName, in))
	}
}

func replaceSequenceLabel(target *core.BuildTarget, label core.BuildLabel, file, in string, runnable, multiple, pairs, dir, outPrefix, test, allOutputs bool) string {
	// Check this label is a dependency of the target, otherwise it's not allowed.
	if label == target.Label { // targets can always use themselves.
		return checkAndReplaceSequence(target, target, file, in, runnable, multiple, pairs, dir, outPrefix, test, allOutputs, false)
	}
	for _, dep := range target.Dependencies {
		if dep.Label == label {
			return checkAndReplaceSequence(target, dep, file, in, runnable, multiple, pairs, dir, outPrefix, test, allOutputs, target.IsTool(label))
		}
	}
	panic(fmt.Sprintf("Rule %s can't use %s; doesn't depend on target %s", target.Label, in, label))
}

func checkAndReplaceSequence(target, dep *core.BuildTarget, file, in string, runnable, multiple, pairs, dir, outPrefix, test, allOutputs, tool bool) string {
	if allOutputs && !multiple && len(dep.Outputs()) != 1 {
		// Label must have only one output.
		panic(fmt.Sprintf("Rule %s can't use %s; %s has multiple outputs.", target.Label, in, dep.Label))
	} else if runnable && !dep.IsBinary {
		panic(fmt.Sprintf("Rule %s can't $(exe %s), it's not executable", target.Label, dep.Label))
	} else if runnable && len(dep.Outputs()) == 0 {
		panic(fmt.Sprintf("Rule %s is tagged as binary but produces no output.", dep.Label))
	}
	output := ""
	for _, out := range dep.Outputs() {
		if file != "" && file != out {
			continue
		}
		if allOutputs || out == in {
			if pairs {
				output += quote(path.Join(core.RepoRoot, dep.OutDir(), out)) + " " + quote(out) + " "
			} else if tool {
				abs, err := filepath.Abs(handleDir(dep.OutDir(), out, dir))
				if err != nil {
					log.Fatalf("Couldn't calculate relative path: %s", err)
				}
				output += quote(abs) + " "
				if dir {
					break
				}
			} else {
				output += quote(fileDestination(target, dep, out, dir, outPrefix, test)) + " "
			}
		}
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
