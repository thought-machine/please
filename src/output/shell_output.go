// Package for displaying output on the command line of the current build state.

package output

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/op/go-logging.v1"

	"build"
	"cli"
	"core"
	"test"
)

var log = logging.MustGetLogger("output")

var startTime = time.Now()

// SetColouredOutput forces on or off coloured output in logging and other console output.
func SetColouredOutput(on bool) {
	cli.StdErrIsATerminal = on
}

// Used to track currently building targets.
type buildingTarget struct {
	sync.Mutex
	buildingTargetData
}

type buildingTargetData struct {
	Label       core.BuildLabel
	Started     time.Time
	Finished    time.Time
	Description string
	Active      bool
	Failed      bool
	Cached      bool
	Err         error
	Colour      string
}

// MonitorState monitors the build while it's running (essentially until state.Results is closed)
// and prints output while it's happening.
func MonitorState(state *core.BuildState, numThreads int, plainOutput, keepGoing, shouldBuild, shouldTest, shouldRun, showStatus bool, traceFile string) bool {
	failedTargetMap := map[core.BuildLabel]error{}
	buildingTargets := make([]buildingTarget, numThreads, numThreads)

	displayDone := make(chan interface{})
	stop := make(chan interface{})
	if !plainOutput {
		go display(state, &buildingTargets, stop, displayDone)
	}
	aggregatedResults := core.TestResults{}
	failedTargets := []core.BuildLabel{}
	failedNonTests := []core.BuildLabel{}
	for result := range state.Results {
		processResult(state, result, buildingTargets, &aggregatedResults, plainOutput, keepGoing, &failedTargets, &failedNonTests, failedTargetMap, traceFile != "")
	}
	if !plainOutput {
		stop <- struct{}{}
		<-displayDone
	}
	if traceFile != "" {
		writeTrace(traceFile)
	}
	duration := time.Since(startTime).Seconds()
	if len(failedNonTests) > 0 { // Something failed in the build step.
		if state.Verbosity > 0 {
			printFailedBuildResults(failedNonTests, failedTargetMap, duration)
		}
		// Die immediately and unsuccessfully, this avoids awkward interactions with
		// --failing_tests_ok later on.
		os.Exit(-1)
	}
	// Check all the targets we wanted to build actually have been built.
	for _, label := range state.ExpandOriginalTargets() {
		if target := state.Graph.Target(label); target == nil {
			log.Fatalf("Target %s doesn't exist in build graph", label)
		} else if (state.NeedHashesOnly || state.PrepareOnly) && target.State() == core.Stopped {
			// Do nothing, we will output about this shortly.
		} else if shouldBuild && target != nil && target.State() < core.Built && len(failedTargetMap) == 0 && !target.AddedPostBuild {
			// N.B. Currently targets that are added post-build are excluded here, because in some legit cases this
			//      check can fail incorrectly. It'd be better to verify this more precisely though.
			cycle := graphCycleMessage(state.Graph, target)
			log.Fatalf("Target %s hasn't built but we have no pending tasks left.\n%s", label, cycle)
		}
	}
	if state.Verbosity > 0 && shouldBuild {
		if shouldTest { // Got to the test phase, report their results.
			printTestResults(state, aggregatedResults, failedTargets, duration)
		} else if state.NeedHashesOnly {
			printHashes(state, duration)
		} else if state.PrepareOnly {
			printTempDirs(state, duration)
		} else if !shouldRun { // Must be plz build or similar, report build outputs.
			printBuildResults(state, duration, showStatus)
		}
	}
	return len(failedTargetMap) == 0
}

func processResult(state *core.BuildState, result *core.BuildResult, buildingTargets []buildingTarget, aggregatedResults *core.TestResults, plainOutput bool,
	keepGoing bool, failedTargets, failedNonTests *[]core.BuildLabel, failedTargetMap map[core.BuildLabel]error, shouldTrace bool) {
	label := result.Label
	active := result.Status == core.PackageParsing || result.Status == core.TargetBuilding || result.Status == core.TargetTesting
	failed := result.Status == core.ParseFailed || result.Status == core.TargetBuildFailed || result.Status == core.TargetTestFailed
	cached := result.Status == core.TargetCached || result.Tests.Cached
	stopped := result.Status == core.TargetBuildStopped
	if shouldTrace {
		addTrace(result, buildingTargets[result.ThreadID].Label, active)
	}
	if failed && result.Tests.NumTests == 0 && result.Tests.Failed == 0 {
		result.Tests.NumTests = 1
		result.Tests.Failed = 1 // Ensure there's one test failure when there're no results to parse.
	}
	// Only aggregate test results the first time it finishes.
	if buildingTargets[result.ThreadID].Active && !active {
		aggregatedResults.Aggregate(&result.Tests)
	}
	target := state.Graph.Target(label)
	updateTarget(state, plainOutput, &buildingTargets[result.ThreadID], label, active, failed, cached, result.Description, result.Err, targetColour(target))
	if failed {
		failedTargetMap[label] = result.Err
		// Don't stop here after test failure, aggregate them for later.
		if !keepGoing && result.Status != core.TargetTestFailed {
			// Reset colour so the entire compiler error output doesn't appear red.
			log.Errorf("%s failed:${RESET}\n%s", result.Label, result.Err)
			state.KillAll()
		} else if !plainOutput { // plain output will have already logged this
			log.Errorf("%s failed: %s", result.Label, result.Err)
		}
		*failedTargets = append(*failedTargets, label)
		if result.Status != core.TargetTestFailed {
			*failedNonTests = append(*failedNonTests, label)
		}
	} else if stopped {
		failedTargetMap[result.Label] = nil
	} else if plainOutput && state.ShowTestOutput && result.Status == core.TargetTested {
		// If using interactive output we'll print it afterwards.
		printf("Finished test %s:\n%s\n", label, target.Results.Output)
	}
}

func printTestResults(state *core.BuildState, aggregatedResults core.TestResults, failedTargets []core.BuildLabel, duration float64) {
	if len(failedTargets) > 0 {
		for _, failed := range failedTargets {
			target := state.Graph.TargetOrDie(failed)
			if len(target.Results.Failures) == 0 {
				if target.Results.TimedOut {
					printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${WHITE_ON_RED}Timed out${RESET}\n", target.Label)
				} else {
					printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${WHITE_ON_RED}Failed to run test${RESET}\n", target.Label)
				}
			} else {
				printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${BOLD_GREEN}%3d passed ${BOLD_YELLOW}%3d skipped ${BOLD_RED}%3d failed ${BOLD_WHITE}Took %3.1fs${RESET}\n",
					target.Label, target.Results.Passed, target.Results.Skipped, target.Results.Failed, target.Results.Duration)
				for _, failure := range target.Results.Failures {
					printf("${BOLD_RED}Failure: %s in %s${RESET}\n", failure.Type, failure.Name)
					printf("%s\n", failure.Traceback)
					if len(failure.Stdout) > 0 {
						printf("${BOLD_RED}Standard output:${RESET}\n%s\n", failure.Stdout)
					}
					if len(failure.Stderr) > 0 {
						printf("${BOLD_RED}Standard error:${RESET}\n%s\n", failure.Stderr)
					}
				}
			}
			if len(target.Results.Output) > 0 {
				printf("${BOLD_RED}Full output:${RESET}\n%s\n", target.Results.Output)
			}
			if target.Results.Flakes > 0 {
				printf("${BOLD_MAGENTA}Flaky target; made %s before giving up${RESET}\n", pluralise(target.Results.Flakes, "attempt", "attempts"))
			}
		}
	}
	// Print individual test results
	i := 0
	for _, target := range state.Graph.AllTargets() {
		if target.IsTest && target.Results.NumTests > 0 {
			if target.Results.Failed > 0 {
				printf("${RED}%s${RESET} %s\n", target.Label, testResultMessage(target.Results, failedTargets))
			} else {
				printf("${GREEN}%s${RESET} %s\n", target.Label, testResultMessage(target.Results, failedTargets))
			}
			if state.ShowTestOutput && target.Results.Output != "" {
				printf("Test output:\n%s\n", target.Results.Output)
			}
			i++
		}
	}
	aggregatedResults.Duration = -0.1 // Exclude this from being displayed later.
	printf(fmt.Sprintf("${BOLD_WHITE}%s and %s${BOLD_WHITE}. Total time %0.2fs.${RESET}\n",
		pluralise(i, "test target", "test targets"), testResultMessage(aggregatedResults, failedTargets), duration))
}

// Produces a string describing the results of one test (or a single aggregation).
func testResultMessage(results core.TestResults, failedTargets []core.BuildLabel) string {
	if results.NumTests == 0 {
		if len(failedTargets) > 0 {
			return "Tests failed"
		}
		return "No tests found"
	}
	msg := fmt.Sprintf("%s run", pluralise(results.NumTests, "test", "tests"))
	if results.Duration >= 0.0 {
		msg += fmt.Sprintf(" in %4.2fs", results.Duration)
	}
	msg += fmt.Sprintf("; ${BOLD_GREEN}%d passed${RESET}", results.Passed)
	if results.Failed > 0 {
		msg += fmt.Sprintf(", ${BOLD_RED}%d failed${RESET}", results.Failed)
	}
	if results.Skipped > 0 {
		msg += fmt.Sprintf(", ${BOLD_YELLOW}%d skipped${RESET}", results.Skipped)
	}
	if results.Flakes > 0 {
		msg += fmt.Sprintf(", ${BOLD_MAGENTA}%s${RESET}", pluralise(results.Flakes, "flake", "flakes"))
	}
	if results.Cached {
		msg += " ${GREEN}[cached]${RESET}"
	}
	return msg
}

func printBuildResults(state *core.BuildState, duration float64, showStatus bool) {
	// Count incrementality.
	totalBuilt := 0
	totalReused := 0
	for _, target := range state.Graph.AllTargets() {
		if target.State() == core.Built {
			totalBuilt++
		} else if target.State() == core.Reused {
			totalReused++
		}
	}
	incrementality := 100.0 * float64(totalReused) / float64(totalBuilt+totalReused)
	if totalBuilt+totalReused == 0 {
		incrementality = 100 // avoid NaN
	}
	// Print this stuff so we always see it.
	printf("Build finished; total time %0.2fs, incrementality %.1f%%. Outputs:\n", duration, incrementality)
	for _, label := range state.ExpandVisibleOriginalTargets() {
		target := state.Graph.TargetOrDie(label)
		if showStatus {
			fmt.Printf("%s [%s]:\n", label, target.State())
		} else {
			fmt.Printf("%s:\n", label)
		}
		for _, result := range buildResult(target) {
			fmt.Printf("  %s\n", result)
		}
	}
}

func printHashes(state *core.BuildState, duration float64) {
	fmt.Printf("Hashes calculated, total time %0.2fs:\n", duration)
	for _, label := range state.ExpandVisibleOriginalTargets() {
		hash, err := build.OutputHash(state.Graph.TargetOrDie(label))
		if err != nil {
			fmt.Printf("  %s: cannot calculate: %s\n", label, err)
		} else {
			fmt.Printf("  %s: %s\n", label, hex.EncodeToString(hash))
		}
	}
}

func printTempDirs(state *core.BuildState, duration float64) {
	fmt.Printf("Temp directories prepared, total time %0.2fs:\n", duration)
	for _, label := range state.ExpandVisibleOriginalTargets() {
		target := state.Graph.TargetOrDie(label)
		cmd := build.ReplaceSequences(target, target.GetCommand())
		env := core.BuildEnvironment(state, target, false)
		fmt.Printf("  %s: %s\n", label, target.TmpDir())
		fmt.Printf("    Command: %s\n", cmd)
		if !state.PrepareShell {
			// This isn't very useful if we're opening a shell (since then the vars will be set anyway)
			fmt.Printf("   Expanded: %s\n", os.Expand(cmd, env.ReplaceEnvironment))
		} else {
			fmt.Printf("\n")
			cmd := exec.Command("bash", "--noprofile", "--norc", "-o", "pipefail") // plz requires bash, some commands contain bashisms.
			cmd.Dir = target.TmpDir()
			cmd.Env = env
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run() // Ignore errors, it will typically end by the user killing it somehow.
		}
	}
}

func buildResult(target *core.BuildTarget) []string {
	results := []string{}
	if target != nil {
		for _, out := range target.Outputs() {
			if core.StartedAtRepoRoot() {
				results = append(results, path.Join(target.OutDir(), out))
			} else {
				results = append(results, path.Join(core.RepoRoot, target.OutDir(), out))
			}
		}
	}
	return results
}

func printFailedBuildResults(failedTargets []core.BuildLabel, failedTargetMap map[core.BuildLabel]error, duration float64) {
	printf("${WHITE_ON_RED}Build stopped after %0.2fs. %s failed:${RESET}\n", duration, pluralise(len(failedTargetMap), "target", "targets"))
	for _, label := range failedTargets {
		err := failedTargetMap[label]
		if err != nil {
			printf("    ${BOLD_RED}%s\n${RESET}%s${RESET}\n", label, colouriseError(err))
		} else {
			printf("    ${BOLD_RED}%s${RESET}\n", label)
		}
	}
}

func updateTarget(state *core.BuildState, plainOutput bool, buildingTarget *buildingTarget, label core.BuildLabel,
	active bool, failed bool, cached bool, description string, err error, colour string) {
	updateTarget2(buildingTarget, label, active, failed, cached, description, err, colour)
	if plainOutput {
		if failed {
			log.Errorf("%s: %s: %s", label.String(), description, err)
		} else {
			if !active {
				active := pluralise(state.NumActive(), "task", "tasks")
				log.Notice("[%d/%s] %s: %s [%3.1fs]", state.NumDone(), active, label.String(), description, time.Now().Sub(buildingTarget.Started).Seconds())
			} else {
				log.Info("%s: %s", label.String(), description)
			}
		}
	}
}

func updateTarget2(target *buildingTarget, label core.BuildLabel, active bool, failed bool, cached bool, description string, err error, colour string) {
	target.Lock()
	defer target.Unlock()
	target.Label = label
	target.Description = description
	if !target.Active {
		// Starting to build now.
		target.Started = time.Now()
		target.Finished = target.Started
	} else if !active {
		// finished building
		target.Finished = time.Now()
	}
	target.Active = active
	target.Failed = failed
	target.Cached = cached
	target.Err = err
	target.Colour = colour
}

func targetColour(target *core.BuildTarget) string {
	if target == nil {
		return "${BOLD_CYAN}" // unknown
	} else if target.IsBinary {
		return "${BOLD}" + targetColour2(target)
	} else {
		return targetColour2(target)
	}
}

func targetColour2(target *core.BuildTarget) string {
	// Quick heuristic on language types. May want to make this configurable.
	for _, require := range target.Requires {
		if require == "py" {
			return "${GREEN}"
		} else if require == "java" {
			return "${RED}"
		} else if require == "go" {
			return "${YELLOW}"
		} else if require == "js" {
			return "${BLUE}"
		}
	}
	if strings.Contains(target.Label.PackageName, "third_party") {
		return "${MAGENTA}"
	}
	return "${WHITE}"
}

// Used to strip the formatting stuff when running directly through 'go run'.
var stripFormatting = regexp.MustCompile("\\$\\{[^\\}]+\\}")

// printf is used throughout this package to print something to stderr with some niceties
// around ANSI formatting codes.
func printf(format string, args ...interface{}) {
	if "${WHITE}"[0] == '$' || !cli.StdErrIsATerminal {
		msg := stripFormatting.ReplaceAllString(fmt.Sprintf(format, args...), "")
		fmt.Fprint(os.Stderr, cli.StripAnsi.ReplaceAllString(msg, ""))
	} else {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// Since this is a gentleman's build tool, we'll make an effort to get plurals correct
// in at least this one place.
func pluralise(num int, singular, plural string) string {
	if num == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", num, plural)
}

// PrintCoverage writes out coverage metrics after a test run in a file tree setup.
// Only files that were covered by tests and not excluded are shown.
func PrintCoverage(state *core.BuildState, includeFiles []string) {
	printf("${BOLD_WHITE}Coverage results:${RESET}\n")
	totalCovered := 0
	totalTotal := 0
	lastDir := "_"
	for _, file := range state.Coverage.OrderedFiles() {
		if !shouldInclude(file, includeFiles) {
			continue
		}
		dir := filepath.Dir(file)
		if dir != lastDir {
			printf("${WHITE}%s:${RESET}\n", strings.TrimRight(dir, "/"))
		}
		lastDir = dir
		covered, total := test.CountCoverage(state.Coverage.Files[file])
		printf("  %s\n", coveragePercentage(covered, total, file[len(dir)+1:]))
		totalCovered += covered
		totalTotal += total
	}
	printf("${BOLD_WHITE}Total coverage: %s${RESET}\n", coveragePercentage(totalCovered, totalTotal, ""))
}

// PrintLineCoverageReport writes out line-by-line coverage metrics after a test run.
func PrintLineCoverageReport(state *core.BuildState, includeFiles []string) {
	coverageColours := map[core.LineCoverage]string{
		core.NotExecutable: "${GREY}",
		core.Unreachable:   "${YELLOW}",
		core.Uncovered:     "${RED}",
		core.Covered:       "${GREEN}",
	}

	printf("${BOLD_WHITE}Covered files:${RESET}\n")
	for _, file := range state.Coverage.OrderedFiles() {
		if !shouldInclude(file, includeFiles) {
			continue
		}
		coverage := state.Coverage.Files[file]
		covered, total := test.CountCoverage(coverage)
		printf("${BOLD_WHITE}%s: %s${RESET}\n", file, coveragePercentage(covered, total, ""))
		f, err := os.Open(file)
		if err != nil {
			printf("${BOLD_RED}Can't open: %s${RESET}\n", err)
			continue
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		i := 0
		for scanner.Scan() {
			if i < len(coverage) {
				printf("${WHITE}%4d %s%s\n", i, coverageColours[coverage[i]], scanner.Text())
			} else {
				// Assume the lines are not executable. This happens for python, for example.
				printf("${WHITE}%4d ${GREY}%s\n", i, scanner.Text())
			}
			i++
		}
		printf("${RESET}\n")
	}
}

// shouldInclude returns true if we should include a file in the coverage display.
func shouldInclude(file string, files []string) bool {
	if len(files) == 0 {
		return true
	}
	for _, f := range files {
		if file == f {
			return true
		}
	}
	return false
}

// Returns some appropriate ANSI colour code for a coverage percentage.
func coverageColour(percentage float32) string {
	// TODO(pebers): consider making these configurable?
	if percentage < 20.0 {
		return "${MAGENTA}"
	} else if percentage < 60.0 {
		return "${BOLD_RED}"
	} else if percentage < 80.0 {
		return "${BOLD_YELLOW}"
	}
	return "${BOLD_GREEN}"
}

func coveragePercentage(covered, total int, label string) string {
	if total == 0 {
		return fmt.Sprintf("${BOLD_MAGENTA}%s No data${RESET}", label)
	}
	percentage := 100.0 * float32(covered) / float32(total)
	return fmt.Sprintf("%s%s %d/%s, %2.1f%%${RESET}", coverageColour(percentage), label, covered, pluralise(total, "line", "lines"), percentage)
}

// colouriseError adds a splash of colour to a compiler error message.
// This is a similar effect to -fcolor-diagnostics in Clang, but we attempt to apply it fairly generically.
func colouriseError(err error) error {
	msg := []string{}
	for _, line := range strings.Split(err.Error(), "\n") {
		if groups := errorMessageRe.FindStringSubmatch(line); groups != nil {
			if groups[3] != "" {
				groups[3] = ", column " + groups[3]
			}
			if groups[4] != "" {
				groups[4] += ": "
			}
			msg = append(msg, fmt.Sprintf("${BOLD_WHITE}%s, line %s%s:${RESET} ${BOLD_RED}%s${RESET}${BOLD_WHITE}%s${RESET}", groups[1], groups[2], groups[3], groups[4], groups[5]))
		} else {
			msg = append(msg, line)
		}
	}
	return fmt.Errorf("%s", strings.Join(msg, "\n"))
}

// errorMessageRe is a regex to find lines that look like they're specifying a file.
var errorMessageRe = regexp.MustCompile(`^([^ ]+\.[^: /]+):([0-9]+):(?:([0-9]+):)? *(?:([a-z-_ ]+):)? (.*)$`)

// graphCycleMessage attempts to detect graph cycles and produces a readable message from it.
func graphCycleMessage(graph *core.BuildGraph, target *core.BuildTarget) string {
	if cycle := findGraphCycle(graph, target); len(cycle) > 0 {
		msg := "Dependency cycle found:\n"
		msg += fmt.Sprintf("    %s\n", cycle[len(cycle)-1].Label)
		for i := len(cycle) - 2; i >= 0; i-- {
			msg += fmt.Sprintf(" -> %s\n", cycle[i].Label)
		}
		msg += fmt.Sprintf(" -> %s\n", cycle[len(cycle)-1].Label)
		return msg + fmt.Sprintf("Sorry, but you'll have to refactor your build files to avoid this cycle.")
	}
	return unbuiltTargetsMessage(graph)
}

// Attempts to detect cycles in the build graph. Returns an empty slice if none is found,
// otherwise returns a slice of labels describing the cycle.
func findGraphCycle(graph *core.BuildGraph, target *core.BuildTarget) []*core.BuildTarget {
	index := func(haystack []*core.BuildTarget, needle *core.BuildTarget) int {
		for i, straw := range haystack {
			if straw == needle {
				return i
			}
		}
		return -1
	}

	done := map[core.BuildLabel]bool{}
	var detectCycle func(*core.BuildTarget, []*core.BuildTarget) []*core.BuildTarget
	detectCycle = func(target *core.BuildTarget, deps []*core.BuildTarget) []*core.BuildTarget {
		if i := index(deps, target); i != -1 {
			return deps[i:]
		} else if done[target.Label] {
			return nil
		}
		done[target.Label] = true
		deps = append(deps, target)
		for _, dep := range target.Dependencies() {
			if cycle := detectCycle(dep, deps); len(cycle) > 0 {
				return cycle
			}
		}
		return nil
	}
	return detectCycle(target, nil)
}

// Prints all targets in the build graph that are marked to be built but not built yet.
func unbuiltTargetsMessage(graph *core.BuildGraph) string {
	msg := ""
	for _, target := range graph.AllTargets() {
		if target.State() == core.Active {
			if graph.AllDepsBuilt(target) {
				msg += fmt.Sprintf("  %s (waiting for deps to build)\n", target.Label)
			} else {
				msg += fmt.Sprintf("  %s\n", target.Label)
			}
		} else if target.State() == core.Pending {
			msg += fmt.Sprintf("  %s (pending build)\n", target.Label)
		}
	}
	if msg != "" {
		return "\nThe following targets have not yet built:\n" + msg
	}
	return ""
}
