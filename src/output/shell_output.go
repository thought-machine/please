// Package for displaying output on the command line of the current build state.

package output

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/peterebden/go-deferred-regex"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/process"
	"github.com/thought-machine/please/src/test"
)

// durationGranularity is the granularity that we build durations at.
const durationGranularity = 10 * time.Millisecond
const testDurationGranularity = time.Millisecond

// MonitorState monitors the build while it's running and prints output until the results
// channel of state has completed.
func MonitorState(state *core.BuildState, plainOutput, detailedTests, streamTestResults, shell, shellRun bool, traceFile string) {
	initPrintf(state.Config)

	if len(state.Config.Please.Motd) != 0 {
		printf("%s\n", state.Config.Please.Motd[rand.IntN(len(state.Config.Please.Motd))])
	}

	var tw *traceWriter
	if traceFile != "" {
		tw = newTraceWriter(traceFile)
		defer tw.Close()
	}

	displayer := setupDisplayer(state, plainOutput)
	t := time.NewTicker(displayer.Frequency())
	defer t.Stop()
	results := state.Results()
	bt := newBuildingTargets(state, plainOutput)
	displayer.Update(bt.Targets())
loop:
	for {
		select {
		case result, ok := <-results:
			if !ok || (state.DebugFailingTests && result.Status == core.TargetTesting) {
				break loop
			}
			if threadID := bt.ProcessResult(result); tw != nil && !result.Status.IsParse() {
				tw.AddTrace(threadID, result, result.Status.IsActive())
			}
			if streamTestResults && (result.Status == core.TargetTested || result.Status == core.TargetTestFailed) {
				os.Stdout.Write(test.SerialiseResultsToXML(state.Graph.TargetOrDie(result.Label), false, state.Config.Test.StoreTestOutputOnSuccess))
				os.Stdout.Write([]byte{'\n'})
			}
		case <-t.C:
			displayer.Update(bt.Targets())
		}
	}
	displayer.Close()

	duration := time.Since(state.StartTime).Round(durationGranularity)
	if len(bt.FailedNonTests) > 0 { // Something failed in the build step.
		printFailedBuildResults(bt.FailedNonTests, bt.FailedTargets, duration)
		return
	}
	if state.NeedBuild {
		// Check all the targets we wanted to build actually have been built.
		for _, label := range state.ExpandOriginalLabels() {
			if target := state.Graph.Target(label); target == nil {
				log.Fatalf("Target %s doesn't exist in build graph", label)
			} else if (state.NeedHashesOnly || state.PrepareOnly || shell) && target.State() == core.Stopped {
				// Do nothing, we will output about this shortly.
			} else if target.State() < core.Built && len(bt.FailedTargets) == 0 && !target.AddedPostBuild {
				log.Fatalf("Target %s hasn't built but we have no pending tasks left.\n%s", label, unbuiltTargetsMessage(state.Graph))
			}
		}
	}
	if state.NeedBuild && len(bt.FailedNonTests) == 0 {
		if state.PrepareOnly || shell {
			printTempDirs(state, duration, shell, shellRun)
		} else if state.NeedTests { // Got to the test phase, report their results.
			printTestResults(state, bt.FailedTargets, duration, detailedTests)
		} else if state.NeedHashesOnly {
			printHashes(state, duration)
		} else if !state.NeedRun { // Must be plz build or similar, report build outputs.
			if !cli.IsATerminal(os.Stdout) {
				printUnformattedBuildResults(state)
			} else {
				printBuildResults(state, duration)
			}
		}
		msgs, totalMessages, actualMessages := cli.CurrentBackend.GetMessageHistory()
		if actualMessages > 0 && !plainOutput {
			printf("Messages:\n")
			for _, msg := range msgs {
				printf("%s\n", msg)
			}
			if totalMessages != actualMessages {
				printf("plus %d more... see plz-out/log/build.log\n", totalMessages-actualMessages)
			}
		}
	}
}

// PrintConnectionMessage prints the message when we're initially connected to a remote server.
func PrintConnectionMessage(url string, targets []core.BuildLabel, tests, coverage bool) {
	printf("${WHITE}Connection established to remote plz server at ${BOLD_WHITE}%s${RESET}.\n", url)
	printf("${WHITE}It's building the following %s: ", pluralise(len(targets), "target", "targets"))
	for i, t := range targets {
		if i > 5 {
			printf("${BOLD_WHITE}...${RESET}")
			break
		} else {
			if i > 0 {
				printf(", ")
			}
			printf("${BOLD_WHITE}%s${RESET}", t)
		}
	}
	printf("\n${WHITE}Running tests: ${BOLD_WHITE}%s${RESET}\n", yesNo(tests))
	printf("${WHITE}Coverage: ${BOLD_WHITE}%s${RESET}\n", yesNo(coverage))
	printf("${BOLD_WHITE}Ctrl+C${RESET}${WHITE} to disconnect from it; that will ${BOLD_WHITE}not${RESET}${WHITE} stop the remote build.${RESET}\n")
}

// PrintDisconnectionMessage prints the message when we're disconnected from the remote server.
func PrintDisconnectionMessage(success, closed, disconnected bool) {
	printf("${BOLD_WHITE}Disconnected from remote plz server.\nStatus: ")
	if disconnected {
		printf("${BOLD_YELLOW}Disconnected${RESET}\n")
	} else if !closed {
		printf("${BOLD_MAGENTA}Unknown${RESET}\n")
	} else if success {
		printf("${BOLD_GREEN}Success${RESET}\n")
	} else {
		printf("${BOLD_RED}Failure${RESET}\n")
	}
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func printTestResults(state *core.BuildState, failedTargets map[core.BuildLabel]error, duration time.Duration, detailed bool) {
	if len(failedTargets) > 0 {
		targets := make(core.BuildLabels, 0, len(failedTargets))
		for t := range failedTargets {
			targets = append(targets, t)
		}
		sort.Sort(targets)
		for _, failed := range targets {
			target := state.Graph.TargetOrDie(failed)
			if target.Test.Results.Failures() == 0 && target.Test.Results.Errors() == 0 {
				if target.Test.Results.TimedOut {
				} else {
					err := failedTargets[failed]
					printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${WHITE_ON_RED}Failed to run test${RESET}: %v\n", target.Label, err)
					target.Test.Results.TestCases = append(target.Test.Results.TestCases, core.TestCase{
						Executions: []core.TestExecution{
							{
								Error: &core.TestResultFailure{
									Type:      "FailedToRun",
									Message:   "Failed to run test",
									Traceback: err.Error(),
								},
							},
						},
					})
				}
			} else {
				results := target.Test.Results
				printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${BOLD_GREEN}%3d passed ${BOLD_YELLOW}%3d skipped ${BOLD_RED}%3d failed ${BOLD_CYAN}%3d errored${RESET} Took ${BOLD_WHITE}%s${RESET}\n",
					target.Label, results.Passes(), results.Skips(), results.Failures(), results.Errors(), results.Duration.Round(durationGranularity))
				for _, failingTestCase := range results.TestCases {
					if failingTestCase.Success() != nil {
						continue
					}
					var execution core.TestExecution
					var failure *core.TestResultFailure
					if failures := failingTestCase.Failures(); len(failures) > 0 {
						execution = failures[0]
						failure = execution.Failure
						printf("${BOLD_RED}Failure${RESET}: ${RED}%s${RESET} in %s\n", failure.Type, failingTestCase.Name)
					} else if errors := failingTestCase.Errors(); len(errors) > 0 {
						execution = errors[0]
						failure = execution.Error
						printf("${BOLD_CYAN}Error${RESET}: ${CYAN}%s${RESET} in %s\n", failure.Type, failingTestCase.Name)
					}
					if failure != nil {
						if failure.Message != "" {
							printf("%s\n", failure.Message)
						}
						printf("%s\n", failure.Traceback)
						if len(execution.Stdout) > 0 {
							printf("${BOLD_RED}Standard output${RESET}:\n%s\n", execution.Stdout)
						}
						if len(execution.Stderr) > 0 {
							printf("${BOLD_RED}Standard error${RESET}:\n%s\n", execution.Stderr)
						}
					}
				}
			}
		}
	}
	// Print individual test results
	targets := []*core.BuildTarget{}
	aggregate := new(core.TestSuite)
	for _, target := range state.Graph.AllTargets() {
		if target.IsTest() && target.Test.Results != nil {
			targets = append(targets, target)
		}
	}
	abbreviateOutput := len(targets) > 100
	for _, target := range targets {
		results := target.Test.Results
		aggregate.TestCases = append(aggregate.TestCases, results.TestCases...)
		aggregate.Duration += results.Duration
		if len(results.TestCases) > 0 {
			if results.Errors() > 0 {
				printf("${CYAN}%s${RESET} %s\n", target.Label, testResultMessage(results, true))
			} else if results.Failures() > 0 {
				printf("${RED}%s${RESET} %s\n", target.Label, testResultMessage(results, true))
			} else if detailed || (len(failedTargets) == 0 && !abbreviateOutput) {
				// Succeeded or skipped
				printf("${GREEN}%s${RESET} %s\n", target.Label, testResultMessage(results, true))
			}
			if state.ShowTestOutput || detailed {
				// Determine max width of test name so we align them
				width := 0
				for _, result := range results.TestCases {
					if len(result.Name) > width {
						width = len(result.Name)
					}
				}
				format := fmt.Sprintf("%%-%ds", width+1)
				for _, result := range results.TestCases {
					printf("    %s\n", formatTestCase(result, fmt.Sprintf(format, result.Name), detailed))
					if len(result.Executions) > 1 {
						for run, execution := range result.Executions {
							printf("        RUN %d: %s\n", run+1, formatTestExecution(execution, detailed))
							if state.ShowTestOutput {
								showExecutionOutput(execution)
							}
						}
					} else if state.ShowTestOutput {
						showExecutionOutput(result.Executions[0])
					}
				}
			}
		} else if results.TimedOut {
			printf("${RED}%s${RESET} ${WHITE_ON_RED}Timed out${RESET}\n", target.Label)
		}
	}
	printf(fmt.Sprintf("${BOLD_WHITE}%s and %s${BOLD_WHITE}.${RESET}\n",
		pluralise(len(targets), "test target", "test targets"), testResultMessage(aggregate, false)))
	printf("${BOLD_WHITE}Total time: %s real, %s compute.${RESET}\n", duration, aggregate.Duration.Round(durationGranularity))
}

func showExecutionOutput(execution core.TestExecution) {
	if execution.Stdout != "" && execution.Stderr != "" {
		printf("StdOut:\n%s\nStdErr:\n%s\n", execution.Stdout, execution.Stderr)
	} else if execution.Stdout != "" {
		print(execution.Stdout)
	} else if execution.Stderr != "" {
		print(execution.Stderr)
	}
}

func formatTestCase(result core.TestCase, name string, detailed bool) string {
	if len(result.Executions) == 0 {
		return formatTestName(result, name) + " (No results)"
	}
	var outcome core.TestExecution
	if len(result.Executions) > 1 && result.Success() != nil {
		return fmt.Sprintf("%s ${BOLD_MAGENTA}%s${RESET}", formatTestName(result, name), "FLAKY PASS")
	}

	if result.Success() != nil {
		outcome = *result.Success()
	} else if result.Skip() != nil {
		outcome = *result.Skip()
	} else if len(result.Errors()) > 0 {
		outcome = result.Errors()[0]
	} else if len(result.Failures()) > 0 {
		outcome = result.Failures()[0]
	}
	return fmt.Sprintf("%s %s", formatTestName(result, name), formatTestExecution(outcome, detailed))
}

func formatTestName(testCase core.TestCase, name string) string {
	if testCase.Success() != nil {
		return fmt.Sprintf("${GREEN}%s${RESET}", name)
	}
	if testCase.Skip() != nil {
		return fmt.Sprintf("${YELLOW}%s${RESET}", name)
	}
	if len(testCase.Errors()) > 0 {
		return fmt.Sprintf("${CYAN}%s${RESET}", name)
	}
	if len(testCase.Failures()) > 0 {
		return fmt.Sprintf("${RED}%s${RESET}", name)
	}
	return testCase.Name
}

func formatTestExecution(execution core.TestExecution, detailed bool) string {
	if execution.Error != nil {
		return "${BOLD_CYAN}ERROR${RESET}"
	}
	if execution.Failure != nil {
		return "${BOLD_RED}FAIL${RESET} " + maybeToString(execution.Duration)
	}
	if execution.Skip != nil {
		if detailed {
			return "${BOLD_YELLOW}SKIP\n        Reason:${RESET} " + execution.Skip.Message
		}
		// Not usually interesting to have a duration when we did no work.
		return "${BOLD_YELLOW}SKIP${RESET}"
	}
	return "${BOLD_GREEN}PASS${RESET} " + maybeToString(execution.Duration)
}

func maybeToString(duration *time.Duration) string {
	if duration == nil {
		return ""
	}
	return fmt.Sprintf(" ${BOLD_WHITE}%s${RESET}", duration.Round(testDurationGranularity))
}

// Produces a string describing the results of one test (or a single aggregation).
func testResultMessage(results *core.TestSuite, showDuration bool) string {
	msg := pluralise(results.Tests(), "test", "tests") + " run"
	if showDuration && results.Duration >= 0.0 {
		msg += fmt.Sprintf(" in ${BOLD_WHITE}%s${RESET}", results.Duration.Round(testDurationGranularity))
	}
	msg += fmt.Sprintf("; ${BOLD_GREEN}%d passed${RESET}", results.Passes())
	if results.Errors() > 0 {
		msg += fmt.Sprintf(", ${BOLD_CYAN}%d errored${RESET}", results.Errors())
	}
	if results.Failures() > 0 {
		msg += fmt.Sprintf(", ${BOLD_RED}%d failed${RESET}", results.Failures())
	}
	if results.Skips() > 0 {
		msg += fmt.Sprintf(", ${BOLD_YELLOW}%d skipped${RESET}", results.Skips())
	}
	if results.FlakyPasses() > 0 {
		msg += fmt.Sprintf(", ${BOLD_MAGENTA}%s${RESET}", pluralise(results.FlakyPasses(), "flake", "flakes"))
	}
	if results.TimedOut {
		msg += ", ${WHITE_ON_RED}TIMED OUT${RESET}"
	}
	if results.Cached {
		msg += " ${GREEN}[cached]${RESET}"
	}
	return msg
}

func printUnformattedBuildResults(state *core.BuildState) {
	for _, label := range state.ExpandVisibleOriginalTargets() {
		for _, result := range buildResult(state.Graph.TargetOrDie(label)) {
			fmt.Printf("%s\n", result)
		}
	}
}

func printBuildResults(state *core.BuildState, duration time.Duration) {
	// Count incrementality.
	totalBuilt := 0
	totalReused := 0
	for _, target := range state.Graph.AllTargets() {
		if s := target.State(); s == core.Built || s == core.BuiltRemotely {
			totalBuilt++
		} else if s == core.Reused || s == core.ReusedRemotely {
			totalReused++
		}
	}
	incrementality := 100.0 * float64(totalReused) / float64(totalBuilt+totalReused)
	if totalBuilt+totalReused == 0 {
		incrementality = 100 // avoid NaN
	}
	// Print this stuff so we always see it.
	printf("Build finished; total time %s, incrementality %.1f%%.", duration, incrementality)
	if state.RemoteClient != nil && state.OutputDownload == core.NoOutputDownload {
		fmt.Printf("\n") // Outputs are not downloaded so do not print them out.
		return
	}
	fmt.Printf(" Outputs:\n")
	for _, label := range state.ExpandVisibleOriginalTargets() {
		target := state.Graph.TargetOrDie(label)
		fmt.Printf("%s:\n", label)
		for _, result := range buildResult(target) {
			fmt.Printf("  %s\n", result)
		}
	}
}

func printHashes(state *core.BuildState, duration time.Duration) {
	fmt.Printf("Hashes calculated, total time %s:\n", duration)
	for _, label := range state.ExpandVisibleOriginalTargets() {
		hash, err := state.TargetHasher.OutputHash(state.Graph.TargetOrDie(label))
		if err != nil {
			fmt.Printf("  %s: cannot calculate: %s\n", label, err)
		} else {
			fmt.Printf("  %s: %s\n", label, hex.EncodeToString(hash))
		}
	}
}

func printTempDirs(state *core.BuildState, duration time.Duration, shell, shellRun bool) {
	fmt.Printf("Temp directories prepared, total time %s:\n", duration)
	state = state.ForArch(state.TargetArch)
	for _, label := range state.ExpandVisibleOriginalTargets() {
		target := state.Graph.TargetOrDie(label)
		cmd := target.GetCommand(state)
		dir := target.TmpDir()
		env := core.StampedBuildEnvironment(state, target, nil, filepath.Join(core.RepoRoot, target.TmpDir()), target.Stamp)
		shouldSandbox := target.Sandbox
		if state.NeedTests {
			cmd = target.GetTestCommand(state)
			dir = filepath.Join(core.RepoRoot, target.TestDir(1))
			env = core.TestEnvironment(state, target, dir, 1)
			shouldSandbox = target.Test.Sandbox
			if len(state.TestArgs) > 0 {
				env["TESTS"] = strings.Join(state.TestArgs, " ")
			}
		}
		cmd, _ = core.ReplaceSequences(state, target, cmd)
		env["CMD"] = cmd
		fmt.Printf("  %s: %s\n", label, dir)
		fmt.Printf("    Command: %s\n", cmd)
		if !shell {
			// This isn't very useful if we're opening a shell (since then the vars will be set anyway)
			fmt.Printf("   Expanded: %s\n", os.Expand(cmd, env.ReplaceEnvironment))
		} else {
			fmt.Printf("\n")
			argv := []string{"bash", "--noprofile", "--norc", "-o", "pipefail"}
			if shellRun {
				argv = append(argv, "-c", cmd)
			}
			log.Debug("Full command: %s", strings.Join(argv, " "))
			cmd := state.ProcessExecutor.ExecCommand(process.NewSandboxConfig(shouldSandbox, shouldSandbox), false, argv[0], argv[1:]...)
			cmd.Dir = dir
			cmd.Env = append(cmd.Env, env.ToSlice()...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			// TODO(jpoole): Read the docs. Attaching stdin and out doesn't seem to work with this.
			cmd.SysProcAttr.Setpgid = false
			cmd.Run() // Ignore errors, it will typically end by the user killing it somehow.
		}
	}
}

func buildResult(target *core.BuildTarget) []string {
	results := []string{}
	if target != nil {
		for _, out := range target.Outputs() {
			if core.StartedAtRepoRoot() {
				results = append(results, filepath.Join(target.OutDir(), out))
			} else {
				results = append(results, filepath.Join(core.RepoRoot, target.OutDir(), out))
			}
		}
	}
	return results
}

func printFailedBuildResults(failedTargets []core.BuildLabel, failedTargetMap map[core.BuildLabel]error, duration time.Duration) {
	printf("${WHITE_ON_RED}Build stopped after %s. %s failed:${RESET}\n", duration, pluralise(len(failedTargetMap), "target", "targets"))
	for _, label := range failedTargets {
		err := failedTargetMap[label]
		if err != nil {
			if cli.ShowColouredOutput {
				printf("    ${BOLD_RED}%s\n${RESET}%s${RESET}\n", label, colouriseError(err))
			} else {
				printf("    %s\n%s\n", label, err)
			}
		} else {
			printf("    ${BOLD_RED}%s${RESET}\n", label)
		}
	}
}

// Since this is a gentleman's build tool, we'll make an effort to get plurals correct
// in at least this one place.
func pluralise(num int, singular, plural string) string {
	if num == 1 {
		return "1 " + singular
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
			if dir == "." {
				printf("${WHITE}top-level:${RESET}\n")
			} else {
				printf("${WHITE}%s:${RESET}\n", strings.TrimRight(dir, "/"))
			}
		}
		lastDir = dir
		covered, total := test.CountCoverage(state.Coverage.Files[file])
		printf("  %s\n", coveragePercentage(covered, total, strings.TrimPrefix(file, dir+"/")))
		totalCovered += covered
		totalTotal += total
	}
	printf("${BOLD_WHITE}Total coverage: %s${RESET}\n", coveragePercentage(totalCovered, totalTotal, ""))
}

// PrintIncrementalCoverage prints the given incremental coverage statistics.
func PrintIncrementalCoverage(stats *test.IncrementalStats) {
	printf("${BOLD_WHITE}Incremental coverage: %s${RESET}\n", coveragePercentage(stats.CoveredLines, stats.ModifiedLines, ""))
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
		if isMatch, _ := filepath.Match(f, file); isMatch {
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
var errorMessageRe = deferredregex.DeferredRegex{Re: `^([^ ]+\.[^: /]+):([0-9]+):(?:([0-9]+):)? *(?:([a-z-_ ]+):)? (.*)$`}

// unbuiltTargetsMessage returns a message for any targets that are supposed to build but haven't yet.
func unbuiltTargetsMessage(graph *core.BuildGraph) string {
	msg := ""
	for _, target := range graph.AllTargets() {
		if target.State() == core.Active {
			msg += fmt.Sprintf("  %s", target.Label)
		} else if target.State() == core.Pending {
			msg += fmt.Sprintf("  %s (pending build)\n", target.Label)
		}
	}
	if msg != "" {
		return "\nThe following targets have not yet built:\n" + msg
	}
	return ""
}

// shortError returns the message for an error, shortening it if the error supports that.
func shortError(err error) string {
	if se, ok := err.(shortenableError); ok {
		return se.ShortError()
	} else if err == nil {
		return "unknown error" // This shouldn't really happen...
	}
	return err.Error()
}

// A shortenableError describes any error type that can communicate a short-form error.
type shortenableError interface {
	ShortError() string
}
