package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"syscall"

	"github.com/jessevdk/go-flags"
	"github.com/kardianos/osext"
	"gopkg.in/op/go-logging.v1"

	"build"
	"cache"
	"clean"
	"core"
	"output"
	"parse"
	"query"
	"run"
	"sync"
	"test"
	"update"
	"utils"
)

var log = logging.MustGetLogger("plz")

var config *core.Configuration

var opts struct {
	BuildFlags struct {
		Config     string   `short:"c" long:"config" description:"Build config to use. Defaults to opt."`
		RepoRoot   string   `short:"r" long:"repo_root" description:"Root of repository to build."`
		KeepGoing  bool     `short:"k" long:"keep_going" description:"Don't stop on first failed target."`
		NumThreads int      `short:"n" long:"num_threads" description:"Number of concurrent build operations. Default is number of CPUs + 2."`
		Include    []string `short:"i" long:"include" description:"Label of targets to include in automatic detection."`
		Exclude    []string `short:"e" long:"exclude" description:"Label of targets to exclude from automatic detection."`
		Engine     string   `long:"engine" hidden:"true" description:"Parser engine .so / .dylib to load"`
	} `group:"Options controlling what to build & how to build it"`

	OutputFlags struct {
		Verbosity         int    `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)" default:"1"`
		LogFile           string `long:"log_file" description:"File to echo full logging output to"`
		LogFileLevel      int    `long:"log_file_level" description:"Log level for file output" default:"2"`
		InteractiveOutput bool   `long:"interactive_output" description:"Show interactive output in a terminal"`
		PlainOutput       bool   `short:"p" long:"plain_output" description:"Don't show interactive output."`
		Colour            bool   `long:"colour" description:"Forces coloured output from logging & other shell output."`
		NoColour          bool   `long:"nocolour" description:"Forces colourless output from logging & other shell output."`
		TraceFile         string `long:"trace_file" description:"File to write Chrome tracing output into"`
		PrintCommands     bool   `long:"print_commands" description:"Print each build / test command as they're run" hidden:"true"`
		Version           bool   `long:"version" description:"Print the version of the tool"`
	} `group:"Options controlling output & logging"`

	FeatureFlags struct {
		NoUpdate           bool `long:"noupdate" description:"Disable Please attempting to auto-update itself."`
		NoCache            bool `long:"nocache" description:"Disable caches (NB. not incrementality)"`
		NoHashVerification bool `long:"nohash_verification" description:"Hash verification errors are nonfatal."`
		NoLock             bool `long:"nolock" description:"Don't attempt to lock the repo exclusively. Use with care."`
		KeepWorkdirs       bool `long:"keep_workdirs" description:"Don't clean directories in plz-out/tmp after successfully building targets."`
	} `group:"Options that enable / disable certain features"`

	AssertVersion    string `long:"assert_version" hidden:"true" description:"Assert the tool matches this version."`
	ParsePackageOnly bool   `description:"Parses a single package only. All that's necessary for some commands." no-flag:"true"`
	NoCacheCleaner   bool   `description:"Don't start a cleaning process for the directory cache" no-flag:"true"`

	Build struct {
		Args struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to build"`
		} `positional-args:"true" required:"true"`
	} `command:"build" description:"Builds one or more targets"`

	Rebuild struct {
		Args struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" required:"true" description:"Targets to rebuild"`
		} `positional-args:"true" required:"true"`
	} `command:"rebuild" description:"Forces a rebuild of one or more targets"`

	Hash struct {
		Args struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to build"`
		} `positional-args:"true" required:"true"`
	} `command:"hash" description:"Calculates hash for one or more targets"`

	Test struct {
		FailingTestsOk  bool   `long:"failing_tests_ok" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)"`
		NumRuns         int    `long:"num_runs" short:"n" description:"Number of times to run each test target."`
		TestResultsFile string `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		// Slightly awkward since we can specify a single test with arguments or multiple test targets.
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors"`
		} `positional-args:"true"`
	} `command:"test" description:"Builds and tests one or more targets"`

	Cover struct {
		FailingTestsOk      bool     `long:"failing_tests_ok" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)"`
		NoCoverageReport    bool     `long:"nocoverage_report" description:"Suppress the per-file coverage report displayed in the shell"`
		LineCoverageReport  bool     `short:"l" long:"line_coverage_report" description:" Show a line-by-line coverage report for all affected files."`
		NumRuns             int      `short:"n" long:"num_runs" description:"Number of times to run each test target."`
		IncludeAllFiles     bool     `short:"a" long:"include_all_files" description:"Include all dependent files in coverage (default is just those from relevant packages)"`
		IncludeFile         []string `long:"include_file" description:"Filenames to filter coverage display to"`
		TestResultsFile     string   `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		CoverageResultsFile string   `long:"coverage_results_file" default:"plz-out/log/coverage.json" description:"File to write combined coverage results to."`
		Args                struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test" group:"one test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors" group:"one test"`
		} `positional-args:"true"`
	} `command:"cover" description:"Builds and tests one or more targets, and calculates coverage."`

	Run struct {
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to run"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments to pass to target when running (to pass flags to the target, put -- before them)"`
		} `positional-args:"true" required:"true"`
	} `command:"run" description:"Builds and runs a single target"`

	Clean struct {
		NoBackground bool     `long:"nobackground" short:"f" description:"Don't fork & detach until clean is finished."`
		Args         struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to clean (default is to clean everything)"`
		} `positional-args:"true"`
	} `command:"clean" description:"Cleans build artifacts" subcommands-optional:"true"`

	Update struct {
	} `command:"update" description:"Checks for an update and updates if needed."`

	Op struct {
	} `command:"op" description:"Re-runs previous command."`

	Init struct {
		Dir                string `long:"dir" description:"Directory to create config in" default:"."`
		BazelCompatibility bool   `long:"bazel_compat" description:"Initialises config for Bazel compatibility mode."`
	} `command:"init" description:"Initialises a .plzconfig file in the current directory"`

	Query struct {
		Deps struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"deps" description:"Queries the dependencies of a target."`
		ReverseDeps struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"reverseDeps" alias:"revdeps" description:"Queries all the reverse dependencies of a target."`
		SomePath struct {
			Args struct {
				Target1 core.BuildLabel `positional-arg-name:"target1" description:"First build target" required:"true"`
				Target2 core.BuildLabel `positional-arg-name:"target2" description:"Second build target" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"somepath" description:"Queries for a path between two targets"`
		AllTargets struct {
			Hidden bool `long:"hidden" description:"Show hidden targets as well"`
			Args   struct {
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
		success, _ := runBuild(opts.Build.Args.Targets, true, false, false)
		return success
	},
	"rebuild": func() bool {
		// It would be more pure to require --nocache for this, but in basically any context that
		// you use 'plz rebuild', you don't want the cache coming in and mucking things up.
		// 'plz clean' followed by 'plz build' would still work in those cases, anyway.
		opts.FeatureFlags.NoCache = true
		success, _ := runBuild(opts.Rebuild.Args.Targets, true, false, false)
		return success
	},
	"hash": func() bool {
		success, _ := runBuild(opts.Hash.Args.Targets, true, false, false)
		return success
	},
	"test": func() bool {
		os.RemoveAll(opts.Test.TestResultsFile)
		targets := testTargets(opts.Test.Args.Target, opts.Test.Args.Args)
		success, state := runBuild(targets, true, true, false)
		test.WriteResultsToFileOrDie(state.Graph, opts.Test.TestResultsFile)
		return success || opts.Test.FailingTestsOk
	},
	"cover": func() bool {
		if opts.BuildFlags.Config != "" {
			log.Warning("Build config overridden; coverage may not be available for some languages")
		} else {
			opts.BuildFlags.Config = "cover"
		}
		os.RemoveAll(opts.Cover.TestResultsFile)
		os.RemoveAll(opts.Cover.CoverageResultsFile)
		targets := testTargets(opts.Cover.Args.Target, opts.Cover.Args.Args)
		success, state := runBuild(targets, true, true, false)
		test.WriteResultsToFileOrDie(state.Graph, opts.Cover.TestResultsFile)
		test.AddOriginalTargetsToCoverage(state, opts.Cover.IncludeAllFiles)
		test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension)
		test.WriteCoverageToFileOrDie(state.Coverage, opts.Cover.CoverageResultsFile)
		if opts.Cover.LineCoverageReport {
			output.PrintLineCoverageReport(state, opts.Cover.IncludeFile)
		} else if !opts.Cover.NoCoverageReport {
			output.PrintCoverage(state, opts.Cover.IncludeFile)
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
			opts.OutputFlags.PlainOutput = true // No need for interactive display, we won't parse anything.
		}
		if success, state := runBuild(opts.Clean.Args.Targets, false, false, true); success {
			if len(opts.Clean.Args.Targets) == 0 {
				state.OriginalTargets = nil // It interprets an empty target list differently.
			}
			clean.Clean(state, state.ExpandOriginalTargets(), !opts.FeatureFlags.NoCache, !opts.Clean.NoBackground)
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
	"reverseDeps": func() bool {
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			state.OriginalTargets = opts.Query.ReverseDeps.Args.Targets
			query.ReverseDeps(state.Graph, state.ExpandOriginalTargets())
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
			query.QueryAllTargets(state.Graph, state.ExpandOriginalTargets(), opts.Query.AllTargets.Hidden)
		})
	},
	"print": func() bool {
		return runQuery(false, opts.Query.Print.Args.Targets, func(state *core.BuildState) {
			query.QueryPrint(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"affectedtargets": func() bool {
		files := opts.Query.AffectedTargets.Args.Files
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			if len(files) == 1 && files[0] == "-" {
				files = utils.ReadAllStdin()
			}
			query.QueryAffectedTargets(state.Graph, files, opts.BuildFlags.Include, opts.BuildFlags.Exclude, opts.Query.AffectedTargets.Tests)
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
			fragments = utils.ReadAllStdin()
		}
		if len(fragments) == 0 || len(fragments) == 1 && strings.Trim(fragments[0], "/ ") == "" {
			os.Exit(0) // Don't do anything for empty completion, it's normally too slow.
		}
		labels := query.QueryCompletionLabels(config, fragments, core.RepoRoot)
		if success, state := Please(labels, config, false, false, false); success {
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
		opts.OutputFlags.PlainOutput = true // No point displaying this for one of these queries.
	}
	if success, state := runBuild(labels, false, false, true); success {
		onSuccess(state)
		return true
	}
	return false
}

func please(tid int, state *core.BuildState, parsePackageOnly bool, include, exclude []string) {
	for {
		label, dependor, t := state.NextTask()
		switch t {
		case core.Stop, core.Kill:
			return
		case core.Parse, core.SubincludeParse:
			parse.Parse(tid, state, label, dependor, parsePackageOnly, include, exclude)
		case core.Build, core.SubincludeBuild:
			build.Build(tid, state, label)
		case core.Test:
			test.Test(tid, state, label)
		}
		state.TaskDone()
	}
}

// Determines from input flags whether we should show 'pretty' output (ie. interactive).
func prettyOutput(interactiveOutput bool, plainOutput bool, verbosity int) bool {
	if interactiveOutput && plainOutput {
		log.Fatal("Can't pass both --interactive_output and --plain_output")
	}
	return interactiveOutput || (!plainOutput && output.StdErrIsATerminal && verbosity < 4)
}

func Please(targets []core.BuildLabel, config *core.Configuration, prettyOutput, shouldBuild, shouldTest bool) (bool, *core.BuildState) {
	if opts.BuildFlags.NumThreads > 0 {
		config.Please.NumThreads = opts.BuildFlags.NumThreads
	} else if config.Please.NumThreads <= 0 {
		config.Please.NumThreads = runtime.NumCPU() + 2
	}
	if opts.NoCacheCleaner {
		config.Cache.DirCacheCleaner = ""
	}
	if opts.BuildFlags.Config != "" {
		config.Build.Config = opts.BuildFlags.Config
	}
	var c *core.Cache
	if !opts.FeatureFlags.NoCache {
		c = cache.NewCache(config)
	}
	state := core.NewBuildState(config.Please.NumThreads, c, opts.OutputFlags.Verbosity, config)
	state.VerifyHashes = !opts.FeatureFlags.NoHashVerification
	state.NumTestRuns = opts.Test.NumRuns + opts.Cover.NumRuns            // Only one of these can be passed.
	state.TestArgs = append(opts.Test.Args.Args, opts.Cover.Args.Args...) // Similarly here.
	state.NeedCoverage = opts.Cover.Args.Target != core.BuildLabel{}
	state.NeedBuild = shouldBuild
	state.NeedTests = shouldTest
	state.NeedHashesOnly = len(opts.Hash.Args.Targets) > 0
	state.PrintCommands = opts.OutputFlags.PrintCommands
	state.CleanWorkdirs = !opts.FeatureFlags.KeepWorkdirs
	state.ForceRebuild = len(opts.Rebuild.Args.Targets) > 0
	state.SetIncludeAndExclude(opts.BuildFlags.Include, opts.BuildFlags.Exclude)
	// Acquire the lock before we start building
	if (shouldBuild || shouldTest) && !opts.FeatureFlags.NoLock {
		core.AcquireRepoLock()
		defer core.ReleaseRepoLock()
	}
	if opts.BuildFlags.Engine != "" {
		state.Config.Please.ParserEngine = opts.BuildFlags.Engine
	}
	// Start looking for the initial targets to kick the build off
	go findOriginalTasks(state, targets)
	// Start up all the build workers
	var wg sync.WaitGroup
	wg.Add(config.Please.NumThreads)
	for i := 0; i < config.Please.NumThreads; i++ {
		go func(tid int) {
			please(tid, state, opts.ParsePackageOnly, opts.BuildFlags.Include, opts.BuildFlags.Exclude)
			wg.Done()
		}(i)
	}
	// Wait until they've all exited, which they'll do once they have no tasks left.
	go func() {
		wg.Wait()
		close(state.Results) // This will signal MonitorState (below) to stop.
	}()
	// Draw stuff to the screen while there are still results coming through.
	success := output.MonitorState(state, config.Please.NumThreads, !prettyOutput, opts.BuildFlags.KeepGoing, shouldBuild, shouldTest, opts.OutputFlags.TraceFile)
	return success, state
}

// findOriginalTasks finds the original parse tasks for the original set of targets.
func findOriginalTasks(state *core.BuildState, targets []core.BuildLabel) {
	for _, target := range targets {
		if target == core.BuildLabelStdin {
			for label := range utils.ReadStdin() {
				findOriginalTask(state, core.ParseBuildLabels([]string{label})[0])
			}
		} else {
			findOriginalTask(state, target)
		}
	}
	state.TaskDone() // initial target adding counts as one.
}

func findOriginalTask(state *core.BuildState, target core.BuildLabel) {
	if target.IsAllSubpackages() {
		for pkg := range utils.FindAllSubpackages(state.Config, target.PackageName, "") {
			state.AddOriginalTarget(core.NewBuildLabel(pkg, "all"))
		}
	} else {
		state.AddOriginalTarget(target)
	}
}

// testTargets handles test targets which can be given in two formats; a list of targets or a single
// target with a list of trailing arguments.
// Alternatively they can be completely omitted in which case we test everything under the working dir.
func testTargets(target core.BuildLabel, args []string) []core.BuildLabel {
	if target.Name == "" {
		return core.InitialPackage()
	} else if len(args) > 0 && core.LooksLikeABuildLabel(args[0]) {
		opts.Cover.Args.Args = []string{}
		opts.Test.Args.Args = []string{}
		return append(core.ParseBuildLabels(args), target)
	} else {
		return []core.BuildLabel{target}
	}
}

// readConfig sets various things up and reads the initial configuration.
func readConfig(forceUpdate bool) *core.Configuration {
	if opts.AssertVersion != "" && core.PleaseVersion != opts.AssertVersion {
		log.Fatalf("Requested Please version %s, but this is version %s", opts.AssertVersion, core.PleaseVersion)
	}
	if opts.FeatureFlags.NoHashVerification {
		log.Warning("You've disabled hash verification; this is intended to help temporarily while modifying build targets. You shouldn't use this regularly.")
	}

	config, err := core.ReadConfigFiles([]string{
		path.Join(core.RepoRoot, core.ConfigFileName),
		path.Join(core.RepoRoot, core.ArchConfigFileName),
		core.MachineConfigFileName,
		path.Join(core.RepoRoot, core.LocalConfigFileName),
	})
	// This is kinda weird, but we need to check for an update before handling errors, because the
	// error may be for a missing config value that we don't know about yet. If we error first it's
	// essentially impossible to add new fields to the config because gcfg doesn't permit unknown fields.
	update.CheckAndUpdate(config, !opts.FeatureFlags.NoUpdate, forceUpdate)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}
	return config
}

// Runs the actual build
// Which phases get run are controlled by shouldBuild and shouldTest.
func runBuild(targets []core.BuildLabel, shouldBuild, shouldTest, defaultToAllTargets bool) (bool, *core.BuildState) {
	if len(targets) == 0 {
		if defaultToAllTargets {
			targets = core.WholeGraph
		} else {
			targets = core.InitialPackage()
		}
	}
	pretty := prettyOutput(opts.OutputFlags.InteractiveOutput, opts.OutputFlags.PlainOutput, opts.OutputFlags.Verbosity)
	return Please(targets, config, pretty, shouldBuild, shouldTest)
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
	parser, extraArgs, flagsErr := output.ParseFlags("Please", &opts, os.Args)
	// Note that we must leave flagsErr for later, because it may be affected by aliases.
	if opts.OutputFlags.Version {
		fmt.Printf("Please version %s\n", core.PleaseVersion)
		os.Exit(0) // Ignore other errors if --version was passed.
	}
	// PrintCommands implies verbosity of at least 2, because commands are logged at that level
	if opts.OutputFlags.PrintCommands && opts.OutputFlags.Verbosity < 2 {
		opts.OutputFlags.Verbosity = 2
	}
	if opts.OutputFlags.Colour {
		output.SetColouredOutput(true)
	} else if opts.OutputFlags.NoColour {
		output.SetColouredOutput(false)
	}
	output.InitLogging(opts.OutputFlags.Verbosity, opts.OutputFlags.LogFile, opts.OutputFlags.LogFileLevel)

	command := activeCommand(parser)
	if command == "init" {
		if flagsErr != nil { // This error otherwise doesn't get checked until later.
			output.ParseFlagsFromArgsOrDie("Please", &opts, os.Args)
		}
		// If we're running plz init then we obviously don't expect to read a config file.
		utils.InitConfig(opts.Init.Dir, opts.Init.BazelCompatibility)
		os.Exit(0)
	}
	if opts.BuildFlags.RepoRoot == "" {
		core.FindRepoRoot(true)
		log.Debug("Found repo root at %s", core.RepoRoot)
	} else {
		core.RepoRoot = opts.BuildFlags.RepoRoot
	}

	// Please always runs from the repo root, so move there now.
	if err := os.Chdir(core.RepoRoot); err != nil {
		log.Fatalf("%s", err)
	}

	config = readConfig(command == "update")

	// Now we've read the config file, we may need to re-run the parser; the aliases in the config
	// can affect how we parse otherwise illegal flag combinations.
	if flagsErr != nil || len(extraArgs) > 0 {
		argv := strings.Join(os.Args, " ")
		for k, v := range config.Aliases {
			argv = strings.Replace(argv, k, v, 1)
		}
		parser = output.ParseFlagsFromArgsOrDie("Please", &opts, strings.Fields(argv))
		command = activeCommand(parser)
	}

	if !buildFunctions[command]() {
		os.Exit(1)
	}
}
