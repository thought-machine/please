// Package for displaying output on the command line of the current build state.

package output

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/test"
)

// durationGranularity is the granularity that we build durations at.
const durationGranularity = 10 * time.Millisecond
const testDurationGranularity = time.Millisecond

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
	Label        core.BuildLabel
	Started      time.Time
	Finished     time.Time
	Description  string
	Active       bool
	Failed       bool
	Cached       bool
	Err          error
	Colour       string
	Target       *core.BuildTarget
	LastProgress float32
	Eta          time.Duration
}

// MonitorState monitors the build while it's running and prints output.
// The caller must cancel the given context once they want this function to stop displaying things.
func MonitorState(ctx context.Context, state *core.BuildState, plainOutput, detailedTests, streamTestResults bool, traceFile string) {
	initPrintf(state.Config)
	failedTargetMap := map[core.BuildLabel]error{}
	buildingTargets := make([]buildingTarget, state.Config.Please.NumThreads+state.Config.NumRemoteExecutors())

	if len(state.Config.Please.Motd) != 0 {
		r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
		printf("%s\n", state.Config.Please.Motd[r.Intn(len(state.Config.Please.Motd))])
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // not really necessary but keeps linter happy
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if plainOutput {
			logProgress(ctx, state, buildingTargets)
		} else {
			display(ctx, state, buildingTargets)
		}
		wg.Done()
	}()
	failedTargets := []core.BuildLabel{}
	failedNonTests := []core.BuildLabel{}
	for result := range state.Results() {
		if state.DebugTests && result.Status == core.TargetTesting {
			cancel() // signals the interactive display goroutines to stop
		}
		processResult(state, result, buildingTargets, plainOutput, &failedTargets, &failedNonTests, failedTargetMap, traceFile != "", streamTestResults)
	}
	<-ctx.Done()
	wg.Wait()
	if traceFile != "" {
		writeTrace(traceFile)
	}
	duration := time.Since(state.StartTime).Round(durationGranularity)
	if len(failedNonTests) > 0 { // Something failed in the build step.
		printFailedBuildResults(failedNonTests, failedTargetMap, duration)
		return
	}
	// Check all the targets we wanted to build actually have been built.
	for _, label := range state.ExpandOriginalTargets() {
		if target := state.Graph.Target(label); target == nil {
			log.Fatalf("Target %s doesn't exist in build graph", label)
		} else if (state.NeedHashesOnly || state.PrepareOnly || state.PrepareShell) && target.State() == core.Stopped {
			// Do nothing, we will output about this shortly.
		} else if state.NeedBuild && target != nil && target.State() < core.Built && len(failedTargetMap) == 0 && !target.AddedPostBuild {
			if err := state.Error(); err != nil {
				log.Fatalf("%s", err)
			}
			// N.B. Currently targets that are added post-build are excluded here, because in some legit cases this
			//      check can fail incorrectly. It'd be better to verify this more precisely though.
			cycle := graphCycleMessage(state.Graph, target)
			log.Fatalf("Target %s hasn't built but we have no pending tasks left.\n%s", label, cycle)
		}
	}
	if state.NeedBuild && len(failedNonTests) == 0 {
		if state.PrepareOnly || state.PrepareShell {
			printTempDirs(state, duration)
		} else if state.NeedTests { // Got to the test phase, report their results.
			printTestResults(state, failedTargets, duration, detailedTests)
		} else if state.NeedHashesOnly {
			printHashes(state, duration)
		} else if !state.NeedRun { // Must be plz build or similar, report build outputs.
			printBuildResults(state, duration)
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

func processResult(state *core.BuildState, result *core.BuildResult, buildingTargets []buildingTarget, plainOutput bool,
	failedTargets, failedNonTests *[]core.BuildLabel, failedTargetMap map[core.BuildLabel]error, shouldTrace, streamTestResults bool) {
	label := result.Label
	active := result.Status.IsActive()
	failed := result.Status.IsFailure()
	cached := result.Status == core.TargetCached || result.Tests.Cached
	stopped := result.Status == core.TargetBuildStopped
	parse := result.Status == core.PackageParsing || result.Status == core.PackageParsed || result.Status == core.ParseFailed
	// Parse events can overlap in weird ways that mess up the display.
	if shouldTrace && !parse {
		addTrace(result, buildingTargets[result.ThreadID].Label, active)
	}
	target := state.Graph.Target(label)
	if !parse { // Parse tasks happen on a different set of threads.
		updateTarget(state, plainOutput, &buildingTargets[result.ThreadID], label, active, failed, cached, result.Description, result.Err, targetColour(target), target)
	}
	if failed {
		failedTargetMap[label] = result.Err
		// Don't stop here after test failure, aggregate them for later.
		if result.Status != core.TargetTestFailed {
			// Reset colour so the entire compiler error output doesn't appear red.
			log.Errorf("%s failed:\x1b[0m\n%s", result.Label, shortError(result.Err))
			state.Stop()
		} else if !plainOutput { // plain output will have already logged this
			log.Errorf("%s failed: %s", result.Label, shortError(result.Err))
		}
		*failedTargets = append(*failedTargets, label)
		if result.Status != core.TargetTestFailed {
			*failedNonTests = append(*failedNonTests, label)
		}
	} else if stopped {
		failedTargetMap[result.Label] = nil
	} else if plainOutput && state.ShowTestOutput && result.Status == core.TargetTested && target != nil {
		// If using interactive output we'll print it afterwards.
		for _, testCase := range target.Results.TestCases {
			printf("Finished test %s:\n", testCase.Name)
			for _, testExecution := range testCase.Executions {
				showExecutionOutput(testExecution)
			}
		}
	}
	if streamTestResults && (result.Status == core.TargetTested || result.Status == core.TargetTestFailed) {
		os.Stdout.Write(test.SerialiseResultsToXML(target, false))
		os.Stdout.Write([]byte{'\n'})
	}
}

func printTestResults(state *core.BuildState, failedTargets []core.BuildLabel, duration time.Duration, detailed bool) {
	if len(failedTargets) > 0 {
		for _, failed := range failedTargets {
			target := state.Graph.TargetOrDie(failed)
			if target.Results.Failures() == 0 && target.Results.Errors() == 0 {
				if target.Results.TimedOut {
				} else {
					printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${WHITE_ON_RED}Failed to run test${RESET}\n", target.Label)
					target.Results.TestCases = append(target.Results.TestCases, core.TestCase{
						Executions: []core.TestExecution{
							{
								Error: &core.TestResultFailure{
									Type:    "FailedToRun",
									Message: "Failed to run test",
								},
							},
						},
					})
				}
			} else {
				printf("${WHITE_ON_RED}Fail:${RED_NO_BG} %s ${BOLD_GREEN}%3d passed ${BOLD_YELLOW}%3d skipped ${BOLD_RED}%3d failed ${BOLD_CYAN}%3d errored${RESET} Took ${BOLD_WHITE}%s${RESET}\n",
					target.Label, target.Results.Passes(), target.Results.Skips(), target.Results.Failures(), target.Results.Errors(), target.Results.Duration.Round(durationGranularity))
				for _, failingTestCase := range target.Results.TestCases {
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
	targets := 0
	aggregate := core.TestSuite{}
	for _, target := range state.Graph.AllTargets() {
		if target.IsTest {
			aggregate.TestCases = append(aggregate.TestCases, target.Results.TestCases...)
			if len(target.Results.TestCases) > 0 {
				if target.Results.Errors() > 0 {
					printf("${CYAN}%s${RESET} %s\n", target.Label, testResultMessage(target.Results))
				} else if target.Results.Failures() > 0 {
					printf("${RED}%s${RESET} %s\n", target.Label, testResultMessage(target.Results))
				} else if detailed || len(failedTargets) == 0 {
					// Succeeded or skipped
					printf("${GREEN}%s${RESET} %s\n", target.Label, testResultMessage(target.Results))
				}
				if state.ShowTestOutput || detailed {
					// Determine max width of test name so we align them
					width := 0
					for _, result := range target.Results.TestCases {
						if len(result.Name) > width {
							width = len(result.Name)
						}
					}
					format := fmt.Sprintf("%%-%ds", width+1)
					for _, result := range target.Results.TestCases {
						printf("    %s\n", formatTestCase(result, fmt.Sprintf(format, result.Name)))
						if len(result.Executions) > 1 {
							for run, execution := range result.Executions {
								printf("        RUN %d: %s\n", run+1, formatTestExecution(execution))
								if state.ShowTestOutput {
									showExecutionOutput(execution)
								}
							}
						} else {
							if state.ShowTestOutput {
								showExecutionOutput(result.Executions[0])
							}
						}
					}
				}
				targets++
			} else if target.Results.TimedOut {
				printf("${RED}%s${RESET} ${WHITE_ON_RED}Timed out${RESET}\n", target.Label)
				targets++
			}
		}
	}
	printf(fmt.Sprintf("${BOLD_WHITE}%s and %s${BOLD_WHITE}. Total time %s.${RESET}\n",
		pluralise(targets, "test target", "test targets"), testResultMessage(aggregate), duration))
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

func formatTestCase(result core.TestCase, name string) string {
	if len(result.Executions) == 0 {
		return fmt.Sprintf("%s (No results)", formatTestName(result, name))
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
	return fmt.Sprintf("%s %s", formatTestName(result, name), formatTestExecution(outcome))
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

func formatTestExecution(execution core.TestExecution) string {
	if execution.Error != nil {
		return "${BOLD_CYAN}ERROR${RESET}"
	}
	if execution.Failure != nil {
		return fmt.Sprintf("${BOLD_RED}FAIL${RESET} %s", maybeToString(execution.Duration))
	}
	if execution.Skip != nil {
		// Not usually interesting to have a duration when we did no work.
		return "${BOLD_YELLOW}SKIP${RESET}"
	}
	return fmt.Sprintf("${BOLD_GREEN}PASS${RESET} %s", maybeToString(execution.Duration))
}

func maybeToString(duration *time.Duration) string {
	if duration == nil {
		return ""
	}
	return fmt.Sprintf(" ${BOLD_WHITE}%s${RESET}", duration.Round(testDurationGranularity))
}

// logProgress continually logs progress messages every 10s explaining where we're up to.
func logProgress(ctx context.Context, state *core.BuildState, buildingTargets []buildingTarget) {
	done := ctx.Done()
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			busy := 0
			for i := 0; i < len(buildingTargets); i++ {
				if buildingTargets[i].Active {
					busy++
				}
			}
			log.Notice("Build running for %s, %d / %d tasks done, %s busy", time.Since(state.StartTime).Round(time.Second), state.NumDone(), state.NumActive(), pluralise(busy, "worker", "workers"))
		case <-done:
			return
		}
	}
}

// Produces a string describing the results of one test (or a single aggregation).
func testResultMessage(results core.TestSuite) string {
	msg := fmt.Sprintf("%s run", pluralise(results.Tests(), "test", "tests"))
	if results.Duration >= 0.0 {
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
		msg += ", ${RED_ON_WHITE}TIMED OUT${RESET}"
	}
	if results.Cached {
		msg += " ${GREEN}[cached]${RESET}"
	}
	return msg
}

func printBuildResults(state *core.BuildState, duration time.Duration) {
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
	printf("Build finished; total time %s, incrementality %.1f%%. Outputs:\n", duration, incrementality)
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

func printTempDirs(state *core.BuildState, duration time.Duration) {
	fmt.Printf("Temp directories prepared, total time %s:\n", duration)
	state = state.ForArch(state.OriginalArch)
	for _, label := range state.ExpandVisibleOriginalTargets() {
		target := state.Graph.TargetOrDie(label)
		cmd := target.GetCommand(state)
		dir := target.TmpDir()
		env := core.StampedBuildEnvironment(state, target, nil, path.Join(core.RepoRoot, target.TmpDir()))
		if state.NeedTests {
			cmd = target.GetTestCommand(state)
			dir = path.Join(core.RepoRoot, target.TestDir())
			env = core.TestEnvironment(state, target, dir)
		}
		cmd, _ = core.ReplaceSequences(state, target, cmd)
		env = append(env, "CMD="+cmd)
		fmt.Printf("  %s: %s\n", label, dir)
		fmt.Printf("    Command: %s\n", cmd)
		if !state.PrepareShell {
			// This isn't very useful if we're opening a shell (since then the vars will be set anyway)
			fmt.Printf("   Expanded: %s\n", os.Expand(cmd, env.ReplaceEnvironment))
		} else {
			fmt.Printf("\n")
			argv := []string{"bash", "--noprofile", "--norc", "-o", "pipefail"}
			if (state.NeedTests && target.TestSandbox) || (!state.NeedTests && target.Sandbox) {
				argv = state.ProcessExecutor.MustSandboxCommand(argv)
			}
			log.Debug("Full command: %s", strings.Join(argv, " "))
			cmd := exec.Command(argv[0], argv[1:]...)
			cmd.Dir = dir
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

func printFailedBuildResults(failedTargets []core.BuildLabel, failedTargetMap map[core.BuildLabel]error, duration time.Duration) {
	printf("${WHITE_ON_RED}Build stopped after %s. %s failed:${RESET}\n", duration, pluralise(len(failedTargetMap), "target", "targets"))
	for _, label := range failedTargets {
		err := failedTargetMap[label]
		if err != nil {
			if cli.StdErrIsATerminal {
				printf("    ${BOLD_RED}%s\n${RESET}%s${RESET}\n", label, colouriseError(err))
			} else {
				printf("    %s\n%s\n", label, err)
			}
		} else {
			printf("    ${BOLD_RED}%s${RESET}\n", label)
		}
	}
}

func updateTarget(state *core.BuildState, plainOutput bool, buildingTarget *buildingTarget, label core.BuildLabel,
	active bool, failed bool, cached bool, description string, err error, colour string, target *core.BuildTarget) {
	updateTarget2(buildingTarget, label, active, failed, cached, description, err, colour, target)
	if plainOutput {
		if !active {
			active := pluralise(state.NumActive(), "task", "tasks")
			log.Info("[%d/%s] %s: %s [%3.1fs]", state.NumDone(), active, label.String(), description, time.Since(buildingTarget.Started).Seconds())
		} else {
			log.Info("%s: %s", label.String(), description)
		}
	}
}

func updateTarget2(target *buildingTarget, label core.BuildLabel, active bool, failed bool, cached bool, description string, err error, colour string, t *core.BuildTarget) {
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
	target.Target = t
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
	for _, require := range target.Requires {
		if colour, present := replacements[require]; present {
			return colour
		}
	}
	return "${WHITE}"
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

// unbuiltTargetsMessage returns a message for any targets that are supposed to build but haven't yet.
func unbuiltTargetsMessage(graph *core.BuildGraph) string {
	msg := ""
	for _, target := range graph.AllTargets() {
		if target.State() == core.Active {
			if target.AllDepsBuilt() {
				msg += fmt.Sprintf("  %s (all deps have built)\n", target.Label)
			} else {
				msg += fmt.Sprintf("  %s (waiting for %s)\n", target.Label, strings.Join(target.UnbuiltDeps(), ", "))
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
