package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"syscall"

	"build"
	"cache"
	"clean"
	"core"
	"output"
	"parse"
	"query"
	"run"
	"test"
	"update"
	"utils"

	"github.com/jessevdk/go-flags"
	"github.com/kardianos/osext"
	"github.com/op/go-logging"
	"golang.org/x/crypto/ssh/terminal"
)

var log = logging.MustGetLogger("plz")

const testResultsFile = "plz-out/log/test_results.xml"
const coverageResultsFile = "plz-out/log/coverage.json"

var Config core.Configuration

var opts struct {
	RepoRoot           string   `short:"r" long:"repo_root" description:"Root of repository to build."`
	NumThreads         int      `short:"n" long:"num_threads" description:"Number of concurrent threads operations."`
	Verbosity          int      `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)" default:"1"`
	KeepGoing          bool     `short:"k" long:"keep_going" description:"Don't stop on first failed target."`
	InteractiveOutput  bool     `long:"interactive_output" description:"Show interactive output in a terminal"`
	PlainOutput        bool     `short:"p" long:"plain_output" description:"Don't show interactive output."`
	Include            []string `short:"i" long:"include" description:"Label of targets to include from automatic detection."`
	Exclude            []string `short:"e" long:"exclude" description:"Label of targets to exclude from automatic detection."`
	NoUpdate           bool     `long:"noupdate" description:"Disable Please attempting to auto-update itself." default:"false"`
	NoCache            bool     `long:"no_cache" description:"Disable caching locally" default:"false"`
	Version            bool     `long:"version" description:"Print the version of the tool"`
	AssertVersion      string   `long:"assert_version" hidden:"true" description:"Assert the tool matches this version."`
	NoHashVerification bool     `long:"nohash_verification" description:"Hash verification errors are nonfatal." default:"false"`
	LogFile            string   `long:"log_file" description:"File to echo full logging output to"`
	LogFileLevel       int      `long:"log_file_level" description:"Log level for file output" default:"2"`
	NoLock             bool     `long:"nolock" description:"Don't attempt to lock the repo exclusively. Use with care." default:"false"`
	TraceFile          string   `long:"trace_file" description:"File to write Chrome tracing output into"`
	PrintCommands      bool     `long:"print_commands" description:"Print each build / test command as they're run"`
	Colour             bool     `long:"colour" description:"Forces coloured output from logging & other shell output."`
	NoColour           bool     `long:"nocolour" description:"Forces colourless output from logging & other shell output."`
	KeepWorkdirs       bool     `long:"keep_workdirs" description:"Don't clean directories in plz-out/tmp after successfully building targets."`

	ParsePackageOnly bool `description:"Parses a single package only. All that's necessary for some commands." no-flag:"true"`
	NoCacheCleaner   bool `description:"Don't start a cleaning process for the directory cache" default:"false" no-flag:"true"`

	Build struct {
		Args struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to build" required:"true"`
		} `positional-args:"true" required:"true"`
	} `command:"build" description:"Builds one or more targets"`

	Test struct {
		FailingTestsOk bool `long:"failing_tests_ok" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)" default:"false"`
		MaxFlakes      int  `long:"max_flakes" description:"Max number of times to run a test (0 for default)"`
		NumRuns        int  `long:"num_runs" short:"n" default:"1" description:"Number of times to run each test target."`
		// Slightly awkward since we can specify a single test with arguments or multiple test targets.
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors"`
		} `positional-args:"true" required:"true"`
	} `command:"test" description:"Builds and tests one or more targets"`

	Cover struct {
		FailingTestsOk   bool `long:"failing_tests_ok" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)" default:"false"`
		NoCoverageReport bool `long:"nocoverage_report" description:"Suppress the per-file coverage report displayed in the shell" default:"false"`
		MaxFlakes        int  `long:"max_flakes" description:"Max number of times to run a test (0 for default)"`
		NumRuns          int  `long:"num_runs" short:"n" default:"1" description:"Number of times to run each test target."`
		Args             struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test" group:"one test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors" group:"one test"`
		} `positional-args:"true" required:"true"`
	} `command:"cover" description:"Builds and tests one or more targets, and calculates coverage."`

	Run struct {
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to run"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments to pass to target when running (to pass flags to the target, put -- before them)"`
		} `positional-args:"true" required:"true"`
	} `command:"run" description:"Builds and runs a single target"`

	Clean struct {
		Cache bool     `long:"cache" short:"c" description:"Clean cache as well" default:"false"`
		NoBackground bool `long:"nobackground" short:"f" description:"Don't fork & detach until clean is finished." default:"false"`
		Args  struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to clean (default is to clean everything)"`
		} `positional-args:"true"`
	} `command:"clean" description:"Cleans build artifacts" subcommands-optional:"true"`

	Update struct {
	} `command:"update" description:"Checks for an update and updates if needed."`

	Op struct {
	} `command:"op" description:"Re-runs previous command."`

	Init struct {
		Dir string `long:"dir" description:"Directory to create config in" default:"."`
	} `command:"init" description:"Initialises a .plzconfig file in the current directory"`

	Query struct {
		Deps struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"deps" description:"Queries the dependencies of a target."`
		SomePath struct {
			Args struct {
				Target1 core.BuildLabel `positional-arg-name:"target1" description:"First build target" required:"true"`
				Target2 core.BuildLabel `positional-arg-name:"target2" description:"Second build target" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"somepath" description:"Queries for a path between two targets"`
		AllTargets struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query"`
			} `positional-args:"true"`
		} `command:"alltargets" description:"Lists all targets in the graph"`
		Print struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to print" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"print" description:"Prints a representation of a single target"`
		Completions struct {
			Cmd  string `long:"cmd" description:"Command to complete for" default:"build"`
			Args struct {
				Fragments []string `positional-arg-name:"fragment" description:"Initial fragment to attempt to complete"`
			} `positional-args:"true"`
		} `command:"completions" description:"Prints possible completions for a string."`
		AffectedTargets struct {
			Tests bool `long:"tests" description:"Shows only affected tests, no other targets."`
			Args  struct {
				Files []string `positional-arg-name:"files" description:"Files to query affected tests for"`
			} `positional-args:"true"`
		} `command:"affectedtargets" description:"Prints any targets affected by a set of files."`
		AffectedTests struct {
			Args struct {
				Files []string `positional-arg-name:"files" description:"Files to query affected tests for"`
			} `positional-args:"true"`
		} `command:"affectedtests" description:"Prints any tests affected by a set of files."`
		Input struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to display inputs for" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"input" description:"Prints all transitive inputs of a target."`
		Output struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to display outputs for" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"output" description:"Prints all outputs of a target."`
		Graph struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to render graph for"`
			} `positional-args:"true"`
		} `command:"graph" description:"Prints a JSON representation of the build graph."`
	} `command:"query" description:"Queries information about the build graph"`
}

// Definitions of what we do for each command.
// Functions are called after args are parsed and return true for success.
var buildFunctions = map[string]func() bool{
	"build": func() bool {
		success, _ := runBuild(opts.Build.Args.Targets, true, false, true)
		return success
	},
	"test": func() bool {
		os.RemoveAll(testResultsFile)
		targets := testTargets(opts.Test.Args.Target, opts.Test.Args.Args)
		success, state := runBuild(targets, true, true, true)
		test.WriteResultsToFileOrDie(state.Graph, testResultsFile)
		return success || opts.Test.FailingTestsOk
	},
	"cover": func() bool {
		os.RemoveAll(testResultsFile)
		os.RemoveAll(coverageResultsFile)
		targets := testTargets(opts.Cover.Args.Target, opts.Cover.Args.Args)
		success, state := runBuild(targets, true, true, true)
		test.WriteResultsToFileOrDie(state.Graph, testResultsFile)
		test.AddOriginalTargetsToCoverage(state, opts.Include, opts.Exclude)
		test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension)
		test.WriteCoverageToFileOrDie(state.Coverage, coverageResultsFile)
		if !opts.Cover.NoCoverageReport {
			output.PrintCoverage(state)
		}
		return success || opts.Cover.FailingTestsOk
	},
	"run": func() bool {
		if success, state := runBuild([]core.BuildLabel{opts.Run.Args.Target}, true, false, false); success {
			run.Run(state.Graph, opts.Run.Args.Target, opts.Run.Args.Args)
		}
		return false // We should never return from run.Run so if we make it here something's wrong.
	},
	"clean": func() bool {
		opts.NoCacheCleaner = true
		if len(opts.Clean.Args.Targets) == 0 {
			opts.PlainOutput = true  // No need for interactive display, we won't parse anything.
		}
		if success, state := runBuild(opts.Clean.Args.Targets, false, false, false); success {
			clean.Clean(state, state.ExpandOriginalTargets(), opts.Clean.Cache, !opts.Clean.NoBackground)
			return true
		}
		return false
	},
	"update": func() bool {
		log.Info("Up to date.")
		return true // We'd have died already if something was wrong.
	},
	"op": func() bool {
		cmd := core.ReadLastOperationOrDie()
		log.Notice("OP PLZ: %s", strings.Join(cmd, " "))
		// Annoyingly we don't seem to have any access to execvp() which would be rather useful here...
		executable, err := osext.Executable()
		if err == nil {
			err = syscall.Exec(executable, append([]string{executable}, cmd...), os.Environ())
		}
		log.Fatalf("SORRY OP: %s", err) // On success Exec never returns.
		return false
	},
	"deps": func() bool {
		return runQuery(true, opts.Query.Deps.Args.Targets, func(state *core.BuildState) {
			query.QueryDeps(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"somepath": func() bool {
		return runQuery(true,
			[]core.BuildLabel{opts.Query.SomePath.Args.Target1, opts.Query.SomePath.Args.Target2},
			func(state *core.BuildState) {
				query.QuerySomePath(state.Graph, opts.Query.SomePath.Args.Target1, opts.Query.SomePath.Args.Target2)
			},
		)
	},
	"alltargets": func() bool {
		return runQuery(true, opts.Query.AllTargets.Args.Targets, func(state *core.BuildState) {
			query.QueryAllTargets(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"print": func() bool {
		return runQuery(false, opts.Query.Print.Args.Targets, func(state *core.BuildState) {
			query.QueryPrint(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"affectedtargets": func() bool {
		files := opts.Query.AffectedTargets.Args.Files
		if len(files) == 1 && files[0] == "-" {
			files = readStdin()
		}
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			query.QueryAffectedTargets(state.Graph, files, opts.Include, opts.Exclude, opts.Query.AffectedTargets.Tests)
		})
	},
	"affectedtests": func() bool {
		// For backwards compatibility.
		// TODO(pebers): Remove at plz 1.2.
		files := opts.Query.AffectedTests.Args.Files
		if len(files) == 1 && files[0] == "-" {
			files = readStdin()
		}
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			query.QueryAffectedTargets(state.Graph, files, opts.Include, opts.Exclude, true)
		})
	},
	"input": func() bool {
		return runQuery(true, opts.Query.Input.Args.Targets, func(state *core.BuildState) {
			query.QueryTargetInputs(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"output": func() bool {
		return runQuery(true, opts.Query.Output.Args.Targets, func(state *core.BuildState) {
			query.QueryTargetOutputs(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"completions": func() bool {
		// Somewhat fiddly because the inputs are not necessarily well-formed at this point.
		opts.ParsePackageOnly = true
		fragments := opts.Query.Completions.Args.Fragments
		if len(fragments) == 1 && fragments[0] == "-" {
			fragments = readStdin()
		} else if len(fragments) == 0 || len(fragments) == 1 && strings.Trim(fragments[0], "/ ") == "" {
			os.Exit(0) // Don't do anything for empty completion, it's normally too slow.
		}
		labels := query.QueryCompletionLabels(Config, fragments, core.RepoRoot)
		if success, state := Please(labels, Config, false, false, false); success {
			binary := opts.Query.Completions.Cmd == "run"
			test := opts.Query.Completions.Cmd == "test" || opts.Query.Completions.Cmd == "cover"
			query.QueryCompletions(state.Graph, labels, binary, test)
			return true
		}
		return false
	},
	"graph": func() bool {
		return runQuery(true, opts.Query.Graph.Args.Targets, func(state *core.BuildState) {
			query.QueryGraph(state.Graph, state.ExpandOriginalTargets())
		})
	},
}

// Used above as a convenience wrapper for query functions.
func runQuery(needFullParse bool, labels []core.BuildLabel, onSuccess func(state *core.BuildState)) bool {
	opts.NoCacheCleaner = true
	if !needFullParse {
		opts.ParsePackageOnly = true
		opts.PlainOutput = true // No point displaying this for one of these queries.
	}
	if success, state := runBuild(labels, false, false, true); success {
		onSuccess(state)
		return true
	}
	return false
}

func please(tid int, state *core.BuildState, parsePackageOnly bool, include, exclude []string) {
	pendingParses, pendingBuilds, pendingTests := state.ReceiveChannels()
	for {
		select {
		case success := <-state.Stop:
			state.Stop <- success // Pass on to another thread
			return
		case pendingParse, ok := <-pendingParses:
			if ok {
				parse.Parse(tid, state, pendingParse.Label, pendingParse.Dependor, parsePackageOnly, include, exclude)
			}
		case pendingBuild, ok := <-pendingBuilds:
			if ok {
				build.Build(tid, state, pendingBuild)
			}
		case pendingTest, ok := <-pendingTests:
			if ok {
				test.Test(tid, state, pendingTest)
			}
		}
		state.ProcessedOne()
	}
}

// Determines from input flags whether we should show 'pretty' output (ie. interactive).
func prettyOutput(interactiveOutput bool, plainOutput bool, verbosity int) bool {
	if interactiveOutput && plainOutput {
		fmt.Printf("Can't pass both --interactive_output and --plain_output\n")
		os.Exit(1)
	}
	return interactiveOutput || (!plainOutput && terminal.IsTerminal(int(os.Stderr.Fd())) && verbosity < 4)
}

func Please(targets []core.BuildLabel, config core.Configuration, prettyOutput, shouldBuild, shouldTest bool) (bool, *core.BuildState) {
	if opts.NumThreads <= 0 {
		opts.NumThreads = runtime.NumCPU() + 2
	}
	if opts.NoCacheCleaner {
		config.Cache.DirCacheCleaner = ""
	}
	var cash *core.Cache = nil
	if !opts.NoCache {
		cash = cache.NewCache(config)
	}
	state := core.NewBuildState(opts.NumThreads, cash, opts.Verbosity, config)
	state.VerifyHashes = !opts.NoHashVerification
	state.MaxFlakes = opts.Test.MaxFlakes + opts.Cover.MaxFlakes          // Only one of these can be passed.
	state.TestArgs = append(opts.Test.Args.Args, opts.Cover.Args.Args...) // Similarly here.
	state.NeedCoverage = opts.Cover.Args.Target != core.BuildLabel{}
	state.NeedBuild = shouldBuild
	state.NeedTests = shouldTest
	state.PrintCommands = opts.PrintCommands
	state.CleanWorkdirs = !opts.KeepWorkdirs
	if opts.Test.NumRuns > opts.Cover.NumRuns {
		state.NumTestRuns = opts.Test.NumRuns
	} else {
		state.NumTestRuns = opts.Cover.NumRuns
	}
	// Acquire the lock before we start building
	if (shouldBuild || shouldTest) && !opts.NoLock {
		core.AcquireRepoLock()
		defer core.ReleaseRepoLock()
	}
	if (shouldBuild || shouldTest) && core.PathExists(testResultsFile) {
		if err := os.Remove(testResultsFile); err != nil {
			log.Fatalf("Failed to remove test results file: %s", err)
		}
	}
	displayDone := make(chan bool)
	go output.MonitorState(state, opts.NumThreads, !prettyOutput, opts.KeepGoing, shouldBuild, shouldTest, displayDone, opts.TraceFile)
	for i := 0; i < opts.NumThreads; i++ {
		go please(i, state, opts.ParsePackageOnly, opts.Include, opts.Exclude)
	}
	for _, target := range targets {
		if target.IsAllSubpackages() {
			for pkg := range parse.FindAllSubpackages(state.Config, target.PackageName, "") {
				label := core.NewBuildLabel(pkg, "all")
				state.OriginalTargets = append(state.OriginalTargets, label)
				state.AddPendingParse(label, core.OriginalTarget)
			}
		} else {
			state.OriginalTargets = append(state.OriginalTargets, target)
			state.AddPendingParse(target, core.OriginalTarget)
		}
	}
	state.ProcessedOne() // initial target adding counts as one.
	state.TargetsLoaded = true
	success := <-state.Stop
	state.Stop <- success
	close(state.Results) // This will signal the output goroutine to stop.
	// TODO(pebers): shouldn't rely on the display routine to tell us whether we succeeded or not...
	success = <-displayDone
	return success, state
}

// Allows input args to come from stdin. It's a little slack to insist on reading it all
// up front but it's too hard to pass around a potential stream of arguments, and none of
// our current use cases really make any difference anyway.
func handleStdinLabels(labels []core.BuildLabel) []core.BuildLabel {
	if len(labels) == 1 && labels[0] == core.BuildLabelStdin {
		return core.ParseBuildLabels(readStdin())
	}
	return labels
}

func readStdin() []string {
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	trimmed := strings.TrimSpace(string(stdin))
	if trimmed == "" {
		log.Warning("No targets supplied; nothing to do.")
		os.Exit(0)
	}
	ret := strings.Split(trimmed, "\n")
	for i, s := range ret {
		ret[i] = strings.TrimSpace(s)
	}
	return ret
}

// Handles test targets which can be given in two formats; a list of targets or a single
// target with a list of trailing arguments.
func testTargets(target core.BuildLabel, args []string) []core.BuildLabel {
	if len(args) > 0 && core.LooksLikeABuildLabel(args[0]) {
		opts.Cover.Args.Args = []string{}
		opts.Test.Args.Args = []string{}
		return append(core.ParseBuildLabels(args), target)
	} else {
		return []core.BuildLabel{target}
	}
}

// Sets various things up and reads the initial configuration.
func readConfig(forceUpdate bool) core.Configuration {
	if opts.AssertVersion != "" && core.PleaseVersion != opts.AssertVersion {
		log.Fatalf("Requested Please version %s, but this is version %s", opts.AssertVersion, core.PleaseVersion)
	}
	if opts.NoHashVerification {
		log.Warning("You've disabled hash verification; this is intended to help temporarily while modifying build targets. You shouldn't use this regularly.")
	}

	config, err := core.ReadConfigFiles([]string{
		path.Join(core.RepoRoot, core.ConfigFileName),
		core.MachineConfigFileName,
		path.Join(core.RepoRoot, core.LocalConfigFileName),
	})
	// This is kinda weird, but we need to check for an update before handling errors, because the
	// error may be for a missing config value that we don't know about yet. If we error first it's
	// essentially impossible to add new fields to the config because gcfg doesn't permit unknown fields.
	update.CheckAndUpdate(config, !opts.NoUpdate, forceUpdate)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}
	return config
}

// Runs the actual build
// Which phases get run are controlled by shouldBuild and shouldTest.
func runBuild(targets []core.BuildLabel, shouldBuild, shouldTest, defaultToAllTargets bool) (bool, *core.BuildState) {
	if len(targets) == 0 && defaultToAllTargets {
		targets = core.WholeGraph
	}
	pretty := prettyOutput(opts.InteractiveOutput, opts.PlainOutput, opts.Verbosity)
	return Please(handleStdinLabels(targets), Config, pretty, shouldBuild, shouldTest)
}

// activeCommand returns the name of the currently active command.
func activeCommand(parser *flags.Parser) string {
	if parser.Active == nil {
		return ""
	} else if parser.Active.Active != nil {
		return parser.Active.Active.Name
	}
	return parser.Active.Name
}

func main() {
	parser, extraArgs, err := output.ParseFlags("Please", &opts, os.Args)
	// PrintCommands implies verbosity of at least 2, because commands are logged at that level
	if opts.PrintCommands && opts.Verbosity < 2 {
		opts.Verbosity = 2
	}
	if opts.Colour {
		output.SetColouredOutput(true)
	} else if opts.NoColour {
		output.SetColouredOutput(false)
	}
	output.InitLogging(opts.Verbosity, opts.LogFile, opts.LogFileLevel)

	command := activeCommand(parser)
	if command == "init" {
		// If we're running plz init then we obviously don't expect to read a config file.
		utils.InitConfig(opts.Init.Dir)
		os.Exit(0)
	}
	if opts.RepoRoot == "" {
		core.FindRepoRoot(true)
		log.Debug("Found repo root at %s", core.RepoRoot)
	} else {
		core.RepoRoot = opts.RepoRoot
	}

	// Please always runs from the repo root, so move there now.
	if err := os.Chdir(core.RepoRoot); err != nil {
		log.Fatalf("%s", err)
	}

	Config = readConfig(command == "update")

	// Now we've read the config file, we may need to re-run the parser; the aliases in the config
	// can affect how we parse otherwise illegal flag combinations.
	if err != nil || len(extraArgs) > 0 {
		argv := strings.Join(os.Args, " ")
		for k, v := range Config.Aliases {
			argv = strings.Replace(argv, k, v, 1)
		}
		parser = output.ParseFlagsFromArgsOrDie("Please", &opts, strings.Fields(argv))
		command = activeCommand(parser)
	}

	if buildFunctions[command]() {
		os.Exit(0)
	}
	os.Exit(1)
}
