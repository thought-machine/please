package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"gopkg.in/op/go-logging.v1"

	"build"
	"cache"
	"clean"
	"cli"
	"core"
	"export"
	"follow"
	"fs"
	"gc"
	"hashes"
	"help"
	"metrics"
	"output"
	"parse"
	"query"
	"run"
	"sync"
	"test"
	"tool"
	"update"
	"utils"
	"watch"
)

var log = logging.MustGetLogger("plz")

var config *core.Configuration

var opts struct {
	Usage      string `usage:"Please is a high-performance multi-language build system.\n\nIt uses BUILD files to describe what to build and how to build it.\nSee https://please.build for more information about how it works and what Please can do for you."`
	BuildFlags struct {
		Config     string          `short:"c" long:"config" description:"Build config to use. Defaults to opt."`
		Arch       cli.Arch        `short:"a" long:"arch" description:"Architecture to compile for."`
		RepoRoot   cli.Filepath    `short:"r" long:"repo_root" description:"Root of repository to build."`
		KeepGoing  bool            `short:"k" long:"keep_going" description:"Don't stop on first failed target."`
		NumThreads int             `short:"n" long:"num_threads" description:"Number of concurrent build operations. Default is number of CPUs + 2."`
		Include    []string        `short:"i" long:"include" description:"Label of targets to include in automatic detection."`
		Exclude    []string        `short:"e" long:"exclude" description:"Label of targets to exclude from automatic detection."`
		Option     ConfigOverrides `short:"o" long:"override" env:"PLZ_OVERRIDES" env-delim:";" description:"Options to override from .plzconfig (e.g. -o please.selfupdate:false)"`
		Profile    string          `long:"profile" env:"PLZ_CONFIG_PROFILE" description:"Configuration profile to load; e.g. --profile=dev will load .plzconfig.dev if it exists."`
	} `group:"Options controlling what to build & how to build it"`

	OutputFlags struct {
		Verbosity         int          `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)" default:"1"`
		LogFile           cli.Filepath `long:"log_file" description:"File to echo full logging output to" default:"plz-out/log/build.log"`
		LogFileLevel      int          `long:"log_file_level" description:"Log level for file output" default:"4"`
		InteractiveOutput bool         `long:"interactive_output" description:"Show interactive output ina  terminal"`
		PlainOutput       bool         `short:"p" long:"plain_output" description:"Don't show interactive output."`
		Colour            bool         `long:"colour" description:"Forces coloured output from logging & other shell output."`
		NoColour          bool         `long:"nocolour" description:"Forces colourless output from logging & other shell output."`
		TraceFile         cli.Filepath `long:"trace_file" description:"File to write Chrome tracing output into"`
		ShowAllOutput     bool         `long:"show_all_output" description:"Show all output live from all commands. Implies --plain_output."`
		CompletionScript  bool         `long:"completion_script" description:"Prints the bash / zsh completion script to stdout"`
		Version           bool         `long:"version" description:"Print the version of the tool"`
	} `group:"Options controlling output & logging"`

	FeatureFlags struct {
		NoUpdate           bool `long:"noupdate" description:"Disable Please attempting to auto-update itself."`
		NoCache            bool `long:"nocache" description:"Disable caches (NB. not incrementality)"`
		NoHashVerification bool `long:"nohash_verification" description:"Hash verification errors are nonfatal."`
		NoLock             bool `long:"nolock" description:"Don't attempt to lock the repo exclusively. Use with care."`
		KeepWorkdirs       bool `long:"keep_workdirs" description:"Don't clean directories in plz-out/tmp after successfully building targets."`
	} `group:"Options that enable / disable certain features"`

	Profile          string `long:"profile_file" hidden:"true" description:"Write profiling output to this file"`
	MemProfile       string `long:"mem_profile_file" hidden:"true" description:"Write a memory profile to this file"`
	ProfilePort      int    `long:"profile_port" hidden:"true" description:"Serve profiling info on this port."`
	ParsePackageOnly bool   `description:"Parses a single package only. All that's necessary for some commands." no-flag:"true"`
	Complete         string `long:"complete" hidden:"true" env:"PLZ_COMPLETE" description:"Provide completion options for this build target."`
	VisibilityParse  bool   `description:"Parse all targets that the original targets are visible to. Used for some query steps." no-flag:"true"`

	Build struct {
		Prepare    bool     `long:"prepare" description:"Prepare build directory for these targets but don't build them."`
		Shell      bool     `long:"shell" description:"Like --prepare, but opens a shell in the build directory with the appropriate environment variables."`
		ShowStatus bool     `long:"show_status" hidden:"true" description:"Show status of each target in output after build"`
		Args       struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to build"`
		} `positional-args:"true" required:"true"`
	} `command:"build" description:"Builds one or more targets"`

	Rebuild struct {
		Args struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" required:"true" description:"Targets to rebuild"`
		} `positional-args:"true" required:"true"`
	} `command:"rebuild" description:"Forces a rebuild of one or more targets"`

	Hash struct {
		Detailed bool `long:"detailed" description:"Produces a detailed breakdown of the hash"`
		Update   bool `short:"u" long:"update" description:"Rewrites the hashes in the BUILD file to the new values"`
		Args     struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to build"`
		} `positional-args:"true" required:"true"`
	} `command:"hash" description:"Calculates hash for one or more targets"`

	Test struct {
		FailingTestsOk  bool         `long:"failing_tests_ok" hidden:"true" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)"`
		NumRuns         int          `long:"num_runs" short:"n" description:"Number of times to run each test target."`
		TestResultsFile cli.Filepath `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		ShowOutput      bool         `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		Debug           bool         `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest, C and C++). Implies -c dbg unless otherwise set."`
		Failed          bool         `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		// Slightly awkward since we can specify a single test with arguments or multiple test targets.
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors"`
		} `positional-args:"true"`
	} `command:"test" description:"Builds and tests one or more targets"`

	Cover struct {
		FailingTestsOk      bool         `long:"failing_tests_ok" hidden:"true" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)"`
		NoCoverageReport    bool         `long:"nocoverage_report" description:"Suppress the per-file coverage report displayed in the shell"`
		LineCoverageReport  bool         `short:"l" long:"line_coverage_report" description:" Show a line-by-line coverage report for all affected files."`
		NumRuns             int          `short:"n" long:"num_runs" description:"Number of times to run each test target."`
		IncludeAllFiles     bool         `short:"a" long:"include_all_files" description:"Include all dependent files in coverage (default is just those from relevant packages)"`
		IncludeFile         []string     `long:"include_file" description:"Filenames to filter coverage display to"`
		TestResultsFile     cli.Filepath `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		CoverageResultsFile cli.Filepath `long:"coverage_results_file" default:"plz-out/log/coverage.json" description:"File to write combined coverage results to."`
		ShowOutput          bool         `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		Debug               bool         `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest, C and C++). Implies -c dbg unless otherwise set."`
		Failed              bool         `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Args                struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test" group:"one test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors" group:"one test"`
		} `positional-args:"true"`
	} `command:"cover" description:"Builds and tests one or more targets, and calculates coverage."`

	Run struct {
		Env      bool `long:"env" description:"Overrides environment variables (e.g. PATH) in the new process."`
		Parallel struct {
			NumTasks       int  `short:"n" long:"num_tasks" default:"10" description:"Maximum number of subtasks to run in parallel"`
			Quiet          bool `short:"q" long:"quiet" description:"Suppress output from successful subprocesses."`
			PositionalArgs struct {
				Targets []core.BuildLabel `positional-arg-name:"target" description:"Targets to run"`
			} `positional-args:"true" required:"true"`
			Args []string `short:"a" long:"arg" description:"Arguments to pass to the called processes."`
		} `command:"parallel" description:"Runs a sequence of targets in parallel"`
		Sequential struct {
			Quiet          bool `short:"q" long:"quiet" description:"Suppress output from successful subprocesses."`
			PositionalArgs struct {
				Targets []core.BuildLabel `positional-arg-name:"target" description:"Targets to run"`
			} `positional-args:"true" required:"true"`
			Args []string `short:"a" long:"arg" description:"Arguments to pass to the called processes."`
		} `command:"sequential" description:"Runs a sequence of targets sequentially."`
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" required:"true" description:"Target to run"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments to pass to target when running (to pass flags to the target, put -- before them)"`
		} `positional-args:"true"`
	} `command:"run" subcommands-optional:"true" description:"Builds and runs a single target"`

	Clean struct {
		NoBackground bool     `long:"nobackground" short:"f" description:"Don't fork & detach until clean is finished."`
		Remote       bool     `long:"remote" description:"Clean entire remote cache when no targets are given (default is local only)"`
		Args         struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to clean (default is to clean everything)"`
		} `positional-args:"true"`
	} `command:"clean" description:"Cleans build artifacts" subcommands-optional:"true"`

	Watch struct {
		Run  bool `short:"r" long:"run" description:"Runs the specified targets when they change (default is to build or test as appropriate)."`
		Args struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" required:"true" description:"Targets to watch the sources of for changes"`
		} `positional-args:"true" required:"true"`
	} `command:"watch" description:"Watches sources of targets for changes and rebuilds them"`

	Update struct {
		Force    bool        `long:"force" description:"Forces a re-download of the new version."`
		NoVerify bool        `long:"noverify" description:"Skips signature verification of downloaded version"`
		Latest   bool        `long:"latest" description:"Update to latest available version (overrides config)."`
		Version  cli.Version `long:"version" description:"Updates to a particular version (overrides config)."`
	} `command:"update" description:"Checks for an update and updates if needed."`

	Op struct {
	} `command:"op" description:"Re-runs previous command."`

	Init struct {
		Dir                cli.Filepath `long:"dir" description:"Directory to create config in" default:"."`
		BazelCompatibility bool         `long:"bazel_compat" description:"Initialises config for Bazel compatibility mode."`
	} `command:"init" description:"Initialises a .plzconfig file in the current directory"`

	Gc struct {
		Conservative bool `short:"c" long:"conservative" description:"Runs a more conservative / safer GC."`
		TargetsOnly  bool `short:"t" long:"targets_only" description:"Only print the targets to delete"`
		SrcsOnly     bool `short:"s" long:"srcs_only" description:"Only print the source files to delete"`
		NoPrompt     bool `short:"y" long:"no_prompt" description:"Remove targets without prompting"`
		DryRun       bool `short:"n" long:"dry_run" description:"Don't remove any targets or files, just print what would be done"`
		Git          bool `short:"g" long:"git" description:"Use 'git rm' to remove unused files instead of just 'rm'."`
		Args         struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to limit gc to."`
		} `positional-args:"true"`
	} `command:"gc" description:"Analyzes the repo to determine unneeded targets."`

	Export struct {
		Output string `short:"o" long:"output" required:"true" description:"Directory to export into"`
		Args   struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to export."`
		} `positional-args:"true"`

		Outputs struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to export."`
			} `positional-args:"true"`
		} `command:"outputs" description:"Exports outputs of a set of targets"`
	} `command:"export" subcommands-optional:"true" description:"Exports a set of targets and files from the repo."`

	Follow struct {
		Retries int          `long:"retries" description:"Number of times to retry the connection"`
		Delay   cli.Duration `long:"delay" default:"1s" description:"Delay between timeouts"`
		Args    struct {
			URL cli.URL `positional-arg-name:"URL" required:"true" description:"URL of remote server to connect to, e.g. 10.23.0.5:7777"`
		} `positional-args:"true"`
	} `command:"follow" description:"Connects to a remote Please instance to stream build events from."`

	Help struct {
		Args struct {
			Topic help.Topic `positional-arg-name:"topic" description:"Topic to display help on"`
		} `positional-args:"true"`
	} `command:"help" alias:"halp" description:"Displays help about various parts of plz or its build rules"`

	Tool struct {
		Args struct {
			Tool tool.Tool `positional-arg-name:"tool" description:"Tool to invoke (jarcat, lint, etc)"`
			Args []string  `positional-arg-name:"arguments" description:"Arguments to pass to the tool"`
		} `positional-args:"true"`
	} `command:"tool" hidden:"true" description:"Invoke one of Please's sub-tools"`

	Query struct {
		Deps struct {
			Unique bool `long:"unique" short:"u" description:"Only output each dependency once"`
			Args   struct {
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
			Fields []string `short:"f" long:"field" description:"Individual fields to print of the target"`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to print" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"print" description:"Prints a representation of a single target"`
		Completions struct {
			Cmd  string `long:"cmd" description:"Command to complete for" default:"build"`
			Args struct {
				Fragments cli.StdinStrings `positional-arg-name:"fragment" description:"Initial fragment to attempt to complete"`
			} `positional-args:"true"`
		} `command:"completions" subcommands-optional:"true" description:"Prints possible completions for a string."`
		AffectedTargets struct {
			Tests        bool `long:"tests" description:"Shows only affected tests, no other targets."`
			Intransitive bool `long:"intransitive" description:"Shows only immediately affected targets, not transitive dependencies."`
			Args         struct {
				Files cli.StdinStrings `positional-arg-name:"files" required:"true" description:"Files to query affected tests for"`
			} `positional-args:"true"`
		} `command:"affectedtargets" description:"Prints any targets affected by a set of files."`
		Input struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to display inputs for" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"input" alias:"inputs" description:"Prints all transitive inputs of a target."`
		Output struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to display outputs for" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"output" alias:"outputs" description:"Prints all outputs of a target."`
		Graph struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to render graph for"`
			} `positional-args:"true"`
		} `command:"graph" description:"Prints a JSON representation of the build graph."`
		WhatOutputs struct {
			EchoFiles bool `long:"echo_files" description:"Echo the file for which the printed output is responsible."`
			Args      struct {
				Files cli.StdinStrings `positional-arg-name:"files" required:"true" description:"Files to query targets responsible for"`
			} `positional-args:"true"`
		} `command:"whatoutputs" description:"Prints out target(s) responsible for outputting provided file(s)"`
		Rules struct {
			Args struct {
				Targets []core.BuildLabel `position-arg-name:"targets" description:"Additional targets to load rules from"`
			} `positional-args:"true"`
		} `command:"rules" description:"Prints built-in rules to stdout as JSON"`
	} `command:"query" description:"Queries information about the build graph"`
}

// Definitions of what we do for each command.
// Functions are called after args are parsed and return true for success.
var buildFunctions = map[string]func() bool{
	"build": func() bool {
		success, _ := runBuild(opts.Build.Args.Targets, true, false)
		return success
	},
	"rebuild": func() bool {
		// It would be more pure to require --nocache for this, but in basically any context that
		// you use 'plz rebuild', you don't want the cache coming in and mucking things up.
		// 'plz clean' followed by 'plz build' would still work in those cases, anyway.
		opts.FeatureFlags.NoCache = true
		success, _ := runBuild(opts.Rebuild.Args.Targets, true, false)
		return success
	},
	"hash": func() bool {
		success, state := runBuild(opts.Hash.Args.Targets, true, false)
		if opts.Hash.Detailed {
			for _, target := range state.ExpandOriginalTargets() {
				build.PrintHashes(state, state.Graph.TargetOrDie(target))
			}
		}
		if opts.Hash.Update {
			hashes.RewriteHashes(state, state.ExpandOriginalTargets())
		}
		return success
	},
	"test": func() bool {
		targets := testTargets(opts.Test.Args.Target, opts.Test.Args.Args, opts.Test.Failed, opts.Test.TestResultsFile)
		os.RemoveAll(string(opts.Test.TestResultsFile))
		success, state := runBuild(targets, true, true)
		test.WriteResultsToFileOrDie(state.Graph, string(opts.Test.TestResultsFile))
		return success || opts.Test.FailingTestsOk
	},
	"cover": func() bool {
		if opts.BuildFlags.Config != "" {
			log.Warning("Build config overridden; coverage may not be available for some languages")
		} else {
			opts.BuildFlags.Config = "cover"
		}
		targets := testTargets(opts.Cover.Args.Target, opts.Cover.Args.Args, opts.Cover.Failed, opts.Cover.TestResultsFile)
		os.RemoveAll(string(opts.Cover.TestResultsFile))
		os.RemoveAll(string(opts.Cover.CoverageResultsFile))
		success, state := runBuild(targets, true, true)
		test.WriteResultsToFileOrDie(state.Graph, string(opts.Cover.TestResultsFile))
		test.AddOriginalTargetsToCoverage(state, opts.Cover.IncludeAllFiles)
		test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension)
		test.WriteCoverageToFileOrDie(state.Coverage, string(opts.Cover.CoverageResultsFile))
		if opts.Cover.LineCoverageReport {
			output.PrintLineCoverageReport(state, opts.Cover.IncludeFile)
		} else if !opts.Cover.NoCoverageReport {
			output.PrintCoverage(state, opts.Cover.IncludeFile)
		}
		return success || opts.Cover.FailingTestsOk
	},
	"run": func() bool {
		if success, state := runBuild([]core.BuildLabel{opts.Run.Args.Target}, true, false); success {
			run.Run(state, opts.Run.Args.Target, opts.Run.Args.Args, opts.Run.Env)
		}
		return false // We should never return from run.Run so if we make it here something's wrong.
	},
	"parallel": func() bool {
		if success, state := runBuild(opts.Run.Parallel.PositionalArgs.Targets, true, false); success {
			os.Exit(run.Parallel(state, state.ExpandOriginalTargets(), opts.Run.Parallel.Args, opts.Run.Parallel.NumTasks, opts.Run.Parallel.Quiet, opts.Run.Env))
		}
		return false
	},
	"sequential": func() bool {
		if success, state := runBuild(opts.Run.Sequential.PositionalArgs.Targets, true, false); success {
			os.Exit(run.Sequential(state, state.ExpandOriginalTargets(), opts.Run.Sequential.Args, opts.Run.Sequential.Quiet, opts.Run.Env))
		}
		return false
	},
	"clean": func() bool {
		config.Cache.DirClean = false
		if len(opts.Clean.Args.Targets) == 0 {
			if len(opts.BuildFlags.Include) == 0 && len(opts.BuildFlags.Exclude) == 0 {
				// Clean everything, doesn't require parsing at all.
				if !opts.Clean.Remote {
					// Don't construct the remote caches if they didn't pass --remote.
					config.Cache.RPCURL = ""
					config.Cache.HTTPURL = ""
				}
				clean.Clean(config, newCache(config), !opts.Clean.NoBackground)
				return true
			}
			opts.Clean.Args.Targets = core.WholeGraph
		}
		if success, state := runBuild(opts.Clean.Args.Targets, false, false); success {
			clean.Targets(state, state.ExpandOriginalTargets(), !opts.FeatureFlags.NoCache)
			return true
		}
		return false
	},
	"watch": func() bool {
		success, state := runBuild(opts.Watch.Args.Targets, false, false)
		if success {
			watch.Watch(state, state.ExpandOriginalTargets(), opts.Watch.Run)
		}
		return success
	},
	"update": func() bool {
		fmt.Printf("Up to date (version %s).\n", core.PleaseVersion)
		return true // We'd have died already if something was wrong.
	},
	"op": func() bool {
		cmd := core.ReadLastOperationOrDie()
		log.Notice("OP PLZ: %s", strings.Join(cmd, " "))
		// Annoyingly we don't seem to have any access to execvp() which would be rather useful here...
		executable, err := os.Executable()
		if err == nil {
			err = syscall.Exec(executable, append([]string{executable}, cmd...), os.Environ())
		}
		log.Fatalf("SORRY OP: %s", err) // On success Exec never returns.
		return false
	},
	"gc": func() bool {
		success, state := runBuild(core.WholeGraph, false, false)
		if success {
			state.OriginalTargets = state.Config.Gc.Keep
			gc.GarbageCollect(state, opts.Gc.Args.Targets, state.ExpandOriginalTargets(), state.Config.Gc.Keep, state.Config.Gc.KeepLabel,
				opts.Gc.Conservative, opts.Gc.TargetsOnly, opts.Gc.SrcsOnly, opts.Gc.NoPrompt, opts.Gc.DryRun, opts.Gc.Git)
		}
		return success
	},
	"export": func() bool {
		success, state := runBuild(opts.Export.Args.Targets, false, false)
		if success {
			export.ToDir(state, opts.Export.Output, state.ExpandOriginalTargets())
		}
		return success
	},
	"follow": func() bool {
		// This is only temporary, ConnectClient will alter it to match the server.
		state := core.NewBuildState(1, nil, opts.OutputFlags.Verbosity, config)
		return follow.ConnectClient(state, opts.Follow.Args.URL.String(), opts.Follow.Retries, time.Duration(opts.Follow.Delay))
	},
	"outputs": func() bool {
		success, state := runBuild(opts.Export.Outputs.Args.Targets, true, false)
		if success {
			export.Outputs(state, opts.Export.Output, state.ExpandOriginalTargets())
		}
		return success
	},
	"help": func() bool {
		return help.Help(string(opts.Help.Args.Topic))
	},
	"tool": func() bool {
		tool.Run(config, opts.Tool.Args.Tool, opts.Tool.Args.Args)
		return false // If the function returns (which it shouldn't), something went wrong.
	},
	"deps": func() bool {
		return runQuery(true, opts.Query.Deps.Args.Targets, func(state *core.BuildState) {
			query.Deps(state, state.ExpandOriginalTargets(), opts.Query.Deps.Unique)
		})
	},
	"reverseDeps": func() bool {
		opts.VisibilityParse = true
		return runQuery(false, opts.Query.ReverseDeps.Args.Targets, func(state *core.BuildState) {
			query.ReverseDeps(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"somepath": func() bool {
		return runQuery(true,
			[]core.BuildLabel{opts.Query.SomePath.Args.Target1, opts.Query.SomePath.Args.Target2},
			func(state *core.BuildState) {
				query.SomePath(state.Graph, opts.Query.SomePath.Args.Target1, opts.Query.SomePath.Args.Target2)
			},
		)
	},
	"alltargets": func() bool {
		return runQuery(true, opts.Query.AllTargets.Args.Targets, func(state *core.BuildState) {
			query.AllTargets(state.Graph, state.ExpandOriginalTargets(), opts.Query.AllTargets.Hidden)
		})
	},
	"print": func() bool {
		return runQuery(false, opts.Query.Print.Args.Targets, func(state *core.BuildState) {
			query.Print(state.Graph, state.ExpandOriginalTargets(), opts.Query.Print.Fields)
		})
	},
	"affectedtargets": func() bool {
		files := opts.Query.AffectedTargets.Args.Files
		targets := core.WholeGraph
		if opts.Query.AffectedTargets.Intransitive {
			state := core.NewBuildState(1, nil, 1, config)
			targets = core.FindOwningPackages(state, files)
		}
		return runQuery(true, targets, func(state *core.BuildState) {
			query.AffectedTargets(state.Graph, files.Get(), opts.BuildFlags.Include, opts.BuildFlags.Exclude, opts.Query.AffectedTargets.Tests, !opts.Query.AffectedTargets.Intransitive)
		})
	},
	"input": func() bool {
		return runQuery(true, opts.Query.Input.Args.Targets, func(state *core.BuildState) {
			query.TargetInputs(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"output": func() bool {
		return runQuery(true, opts.Query.Output.Args.Targets, func(state *core.BuildState) {
			query.TargetOutputs(state.Graph, state.ExpandOriginalTargets())
		})
	},
	"completions": func() bool {
		// Somewhat fiddly because the inputs are not necessarily well-formed at this point.
		opts.ParsePackageOnly = true
		fragments := opts.Query.Completions.Args.Fragments.Get()
		if opts.Query.Completions.Cmd == "help" {
			// Special-case completing help topics rather than build targets.
			if len(fragments) == 0 {
				help.Topics("")
			} else {
				help.Topics(fragments[0])
			}
			return true
		}
		if len(fragments) == 0 || len(fragments) == 1 && strings.Trim(fragments[0], "/ ") == "" {
			os.Exit(0) // Don't do anything for empty completion, it's normally too slow.
		}
		labels, parseLabels, hidden := query.CompletionLabels(config, fragments, core.RepoRoot)
		if success, state := Please(parseLabels, config, false, false, false); success {
			binary := opts.Query.Completions.Cmd == "run"
			test := opts.Query.Completions.Cmd == "test" || opts.Query.Completions.Cmd == "cover"
			query.Completions(state.Graph, labels, binary, test, hidden)
			return true
		}
		return false
	},
	"graph": func() bool {
		return runQuery(true, opts.Query.Graph.Args.Targets, func(state *core.BuildState) {
			if len(opts.Query.Graph.Args.Targets) == 0 {
				state.OriginalTargets = opts.Query.Graph.Args.Targets // It special-cases doing the full graph.
			}
			query.Graph(state, state.ExpandOriginalTargets())
		})
	},
	"whatoutputs": func() bool {
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			query.WhatOutputs(state.Graph, opts.Query.WhatOutputs.Args.Files.Get(), opts.Query.WhatOutputs.EchoFiles)
		})
	},
	"rules": func() bool {
		targets := opts.Query.Rules.Args.Targets
		success, state := Please(opts.Query.Rules.Args.Targets, config, true, true, false)
		if !success {
			return false
		}
		targets = state.ExpandOriginalTargets()
		parse.PrintRuleArgs(state, targets)
		return true
	},
}

// ConfigOverrides are used to implement completion on the -o flag.
type ConfigOverrides map[string]string

// Complete implements the flags.Completer interface.
func (overrides ConfigOverrides) Complete(match string) []flags.Completion {
	return core.DefaultConfiguration().Completions(match)
}

// Used above as a convenience wrapper for query functions.
func runQuery(needFullParse bool, labels []core.BuildLabel, onSuccess func(state *core.BuildState)) bool {
	opts.OutputFlags.PlainOutput = true // No point displaying this for one of these queries.
	config.Cache.DirClean = false
	if !needFullParse {
		opts.ParsePackageOnly = true
	}
	if len(labels) == 0 {
		labels = core.WholeGraph
	}
	if success, state := runBuild(labels, false, false); success {
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
			t := t
			label := label
			dependor := dependor
			state.ParsePool <- func() {
				parse.Parse(tid, state, label, dependor, parsePackageOnly, include, exclude, t == core.SubincludeParse)
				if opts.VisibilityParse && state.IsOriginalTarget(label) {
					parseForVisibleTargets(state, label)
				}
				state.TaskDone()
			}
		case core.Build, core.SubincludeBuild:
			build.Build(tid, state, label)
			state.TaskDone()
		case core.Test:
			test.Test(tid, state, label)
			state.TaskDone()
		}
	}
}

// parseForVisibleTargets adds parse tasks for any targets that the given label is visible to.
func parseForVisibleTargets(state *core.BuildState, label core.BuildLabel) {
	if target := state.Graph.Target(label); target != nil {
		for _, vis := range target.Visibility {
			findOriginalTask(state, vis, false)
		}
	}
}

// prettyOutputs determines from input flags whether we should show 'pretty' output (ie. interactive).
func prettyOutput(interactiveOutput bool, plainOutput bool, verbosity int) bool {
	if interactiveOutput && plainOutput {
		log.Fatal("Can't pass both --interactive_output and --plain_output")
	}
	return interactiveOutput || (!plainOutput && cli.StdErrIsATerminal && verbosity < 4)
}

// newCache constructs a new cache based on the current config / flags.
func newCache(config *core.Configuration) core.Cache {
	if opts.FeatureFlags.NoCache {
		return nil
	}
	return cache.NewCache(config)
}

// Please starts & runs the main build process through to its completion.
func Please(targets []core.BuildLabel, config *core.Configuration, prettyOutput, shouldBuild, shouldTest bool) (bool, *core.BuildState) {
	if opts.BuildFlags.NumThreads > 0 {
		config.Please.NumThreads = opts.BuildFlags.NumThreads
	} else if config.Please.NumThreads <= 0 {
		config.Please.NumThreads = runtime.NumCPU() + 2
	}
	debugTests := opts.Test.Debug || opts.Cover.Debug
	if opts.BuildFlags.Config != "" {
		config.Build.Config = opts.BuildFlags.Config
	} else if debugTests {
		config.Build.Config = "dbg"
	}
	c := newCache(config)
	state := core.NewBuildState(config.Please.NumThreads, c, opts.OutputFlags.Verbosity, config)
	state.VerifyHashes = !opts.FeatureFlags.NoHashVerification
	state.NumTestRuns = opts.Test.NumRuns + opts.Cover.NumRuns            // Only one of these can be passed.
	state.TestArgs = append(opts.Test.Args.Args, opts.Cover.Args.Args...) // Similarly here.
	state.NeedCoverage = !opts.Cover.Args.Target.IsEmpty()
	state.NeedBuild = shouldBuild
	state.NeedTests = shouldTest
	state.NeedHashesOnly = len(opts.Hash.Args.Targets) > 0
	state.PrepareOnly = opts.Build.Prepare || opts.Build.Shell
	state.PrepareShell = opts.Build.Shell
	state.CleanWorkdirs = !opts.FeatureFlags.KeepWorkdirs
	state.ForceRebuild = len(opts.Rebuild.Args.Targets) > 0
	state.ShowTestOutput = opts.Test.ShowOutput || opts.Cover.ShowOutput
	state.DebugTests = debugTests
	state.ShowAllOutput = opts.OutputFlags.ShowAllOutput
	state.SetIncludeAndExclude(opts.BuildFlags.Include, opts.BuildFlags.Exclude)
	parse.InitParser(state)
	if config.Events.Port != 0 && shouldBuild {
		shutdown := follow.InitialiseServer(state, config.Events.Port)
		defer shutdown()
	}
	if config.Events.Port != 0 || config.Display.SystemStats {
		go follow.UpdateResources(state)
	}
	metrics.InitFromConfig(config)
	// Acquire the lock before we start building
	if (shouldBuild || shouldTest) && !opts.FeatureFlags.NoLock {
		core.AcquireRepoLock()
		defer core.ReleaseRepoLock()
	}
	if state.DebugTests && len(targets) != 1 {
		log.Fatalf("-d/--debug flag can only be used with a single test target")
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
	shouldRun := !opts.Run.Args.Target.IsEmpty()
	success := output.MonitorState(state, config.Please.NumThreads, !prettyOutput, opts.BuildFlags.KeepGoing, shouldBuild, shouldTest, shouldRun, opts.Build.ShowStatus, string(opts.OutputFlags.TraceFile))
	metrics.Stop()
	build.StopWorkers()
	if c != nil {
		c.Shutdown()
	}
	return success, state
}

// findOriginalTasks finds the original parse tasks for the original set of targets.
func findOriginalTasks(state *core.BuildState, targets []core.BuildLabel) {
	if state.Config.Bazel.Compatibility && fs.FileExists("WORKSPACE") {
		// We have to parse the WORKSPACE file before anything else to understand subrepos.
		// This is a bit crap really since it inhibits parallelism for the first step.
		parse.Parse(0, state, core.NewBuildLabel("workspace", "all"), core.OriginalTarget, false, state.Include, state.Exclude, false)
	}
	if opts.BuildFlags.Arch.Arch != "" {
		// Set up a new subrepo for this architecture.
		state.Graph.AddSubrepo(core.SubrepoForArch(state, opts.BuildFlags.Arch))
	}
	for _, target := range targets {
		if target == core.BuildLabelStdin {
			for label := range cli.ReadStdin() {
				findOriginalTask(state, core.ParseBuildLabels([]string{label})[0], true)
			}
		} else {
			findOriginalTask(state, target, true)
		}
	}
	state.TaskDone() // initial target adding counts as one.
}

func findOriginalTask(state *core.BuildState, target core.BuildLabel, addToList bool) {
	if opts.BuildFlags.Arch.Arch != "" {
		target.PackageName = path.Join(opts.BuildFlags.Arch.String(), target.PackageName)
	}
	if target.IsAllSubpackages() {
		for pkg := range utils.FindAllSubpackages(state.Config, target.PackageName, "") {
			state.AddOriginalTarget(core.NewBuildLabel(pkg, "all"), addToList)
		}
	} else {
		state.AddOriginalTarget(target, addToList)
	}
}

// testTargets handles test targets which can be given in two formats; a list of targets or a single
// target with a list of trailing arguments.
// Alternatively they can be completely omitted in which case we test everything under the working dir.
// One can also pass a 'failed' flag which runs the failed tests from last time.
func testTargets(target core.BuildLabel, args []string, failed bool, resultsFile cli.Filepath) []core.BuildLabel {
	if failed {
		targets, args := test.LoadPreviousFailures(string(resultsFile))
		// Have to reset these - it doesn't matter which gets which.
		opts.Test.Args.Args = args
		opts.Cover.Args.Args = nil
		return targets
	} else if target.Name == "" {
		return core.InitialPackage()
	} else if len(args) > 0 && core.LooksLikeABuildLabel(args[0]) {
		opts.Cover.Args.Args = []string{}
		opts.Test.Args.Args = []string{}
		return append(core.ParseBuildLabels(args), target)
	}
	return []core.BuildLabel{target}
}

// readConfig sets various things up and reads the initial configuration.
func readConfig(forceUpdate bool) *core.Configuration {
	if opts.FeatureFlags.NoHashVerification {
		log.Warning("You've disabled hash verification; this is intended to help temporarily while modifying build targets. You shouldn't use this regularly.")
	}

	config, err := core.ReadConfigFiles([]string{
		core.MachineConfigFileName,
		core.ExpandHomePath(core.UserConfigFileName),
		path.Join(core.RepoRoot, core.ConfigFileName),
		path.Join(core.RepoRoot, core.ArchConfigFileName),
		path.Join(core.RepoRoot, core.LocalConfigFileName),
	}, opts.BuildFlags.Profile)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	} else if err := config.ApplyOverrides(opts.BuildFlags.Option); err != nil {
		log.Fatalf("Can't override requested config setting: %s", err)
	}
	// Now apply any flags that override this
	if opts.Update.Latest {
		config.Please.Version.Unset()
	} else if opts.Update.Version.IsSet {
		config.Please.Version = opts.Update.Version
	}
	update.CheckAndUpdate(config, !opts.FeatureFlags.NoUpdate, forceUpdate, opts.Update.Force, !opts.Update.NoVerify)
	return config
}

// Runs the actual build
// Which phases get run are controlled by shouldBuild and shouldTest.
func runBuild(targets []core.BuildLabel, shouldBuild, shouldTest bool) (bool, *core.BuildState) {
	if len(targets) == 0 {
		targets = core.InitialPackage()
	}
	pretty := prettyOutput(opts.OutputFlags.InteractiveOutput, opts.OutputFlags.PlainOutput, opts.OutputFlags.Verbosity)
	return Please(targets, config, pretty, shouldBuild, shouldTest)
}

// activeCommand returns the name of the currently active command.
func activeCommand(command *flags.Command) string {
	if command.Active != nil {
		return activeCommand(command.Active)
	}
	return command.Name
}

// readConfigAndSetRoot reads the .plzconfig files and moves to the repo root.
func readConfigAndSetRoot(forceUpdate bool) *core.Configuration {
	if opts.BuildFlags.RepoRoot == "" {
		log.Debug("Found repo root at %s", core.MustFindRepoRoot())
	} else {
		core.RepoRoot = string(opts.BuildFlags.RepoRoot)
	}

	// Please always runs from the repo root, so move there now.
	if err := os.Chdir(core.RepoRoot); err != nil {
		log.Fatalf("%s", err)
	}
	// Reset this now we're at the repo root.
	if opts.OutputFlags.LogFile != "" {
		if !path.IsAbs(string(opts.OutputFlags.LogFile)) {
			opts.OutputFlags.LogFile = cli.Filepath(path.Join(core.RepoRoot, string(opts.OutputFlags.LogFile)))
		}
		cli.InitFileLogging(string(opts.OutputFlags.LogFile), opts.OutputFlags.LogFileLevel)
	}

	return readConfig(forceUpdate)
}

// handleCompletions handles shell completion. Typically it just prints to stdout but
// may do a little more if we think we need to handle aliases.
func handleCompletions(parser *flags.Parser, items []flags.Completion) {
	if len(items) > 0 {
		printCompletions(items)
	} else {
		cli.InitLogging(0)                // Ensure this is quiet
		opts.FeatureFlags.NoUpdate = true // Ensure we don't try to update
		config := readConfigAndSetRoot(false)
		if len(config.Aliases) > 0 {
			for k, v := range config.Aliases {
				parser.AddCommand(k, v, v, &struct{}{})
			}
			// Run again without this registered as a completion handler
			parser.CompletionHandler = nil
			parser.ParseArgs(os.Args[1:])
		}
	}
	// Regardless of what happened, always exit with 0 at this point.
	os.Exit(0)
}

// printCompletions prints a set of completions to stdout.
func printCompletions(items []flags.Completion) {
	for _, item := range items {
		fmt.Println(item.Item)
	}
}

func main() {
	parser, extraArgs, flagsErr := cli.ParseFlags("Please", &opts, os.Args, handleCompletions)
	// Note that we must leave flagsErr for later, because it may be affected by aliases.
	if opts.OutputFlags.Version {
		fmt.Printf("Please version %s\n", core.PleaseVersion)
		os.Exit(0) // Ignore other flags if --version was passed.
	}
	if opts.OutputFlags.Colour {
		output.SetColouredOutput(true)
	} else if opts.OutputFlags.NoColour {
		output.SetColouredOutput(false)
	}
	if opts.OutputFlags.ShowAllOutput {
		opts.OutputFlags.PlainOutput = true
	}
	// Init logging, but don't do file output until we've chdir'd.
	cli.InitLogging(opts.OutputFlags.Verbosity)

	command := activeCommand(parser.Command)
	if opts.Complete != "" {
		// Completion via PLZ_COMPLETE env var sidesteps other commands
		opts.Query.Completions.Cmd = command
		opts.Query.Completions.Args.Fragments = []string{opts.Complete}
		command = "completions"
	} else if command == "init" {
		if flagsErr != nil { // This error otherwise doesn't get checked until later.
			cli.ParseFlagsFromArgsOrDie("Please", core.PleaseVersion.String(), &opts, os.Args)
		}
		// If we're running plz init then we obviously don't expect to read a config file.
		utils.InitConfig(string(opts.Init.Dir), opts.Init.BazelCompatibility)
		os.Exit(0)
	} else if command == "help" || command == "follow" {
		config = core.DefaultConfiguration()
		if !buildFunctions[command]() {
			os.Exit(1)
		}
		os.Exit(0)
	} else if opts.OutputFlags.CompletionScript {
		utils.PrintCompletionScript()
		os.Exit(0)
	}
	// Read the config now
	config = readConfigAndSetRoot(command == "update")
	// Set this in case anything wants to use it soon
	core.NewBuildState(config.Please.NumThreads, nil, opts.OutputFlags.Verbosity, config)

	// Now we've read the config file, we may need to re-run the parser; the aliases in the config
	// can affect how we parse otherwise illegal flag combinations.
	if flagsErr != nil || len(extraArgs) > 0 {
		for idx, arg := range os.Args[1:] {
			// Please should not touch anything that comes after `--`
			if arg == "--" {
				break
			}
			for k, v := range config.Aliases {
				if arg == k {
					// We could insert every token in v into os.Args at this point and then we could have
					// aliases defined in terms of other aliases but that seems rather like overkill so just
					// stick the replacement in wholesale instead.
					os.Args[idx+1] = v
				}
			}
		}
		argv := strings.Join(os.Args[1:], " ")
		parser = cli.ParseFlagsFromArgsOrDie("Please", core.PleaseVersion.String(), &opts, strings.Fields(os.Args[0]+" "+argv))
		command = activeCommand(parser.Command)
	}

	if opts.ProfilePort != 0 {
		go func() {
			log.Warning("%s", http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", opts.ProfilePort), nil))
		}()
	}
	if opts.Profile != "" {
		f, err := os.Create(opts.Profile)
		if err != nil {
			log.Fatalf("Failed to open profile file: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start profiler: %s", err)
		}
		defer pprof.StopCPUProfile()
	}
	if opts.MemProfile != "" {
		f, err := os.Create(opts.MemProfile)
		if err != nil {
			log.Fatalf("Failed to open memory profile file: %s", err)
		}
		defer f.Close()
		defer pprof.WriteHeapProfile(f)
	}

	if !buildFunctions[command]() {
		os.Exit(7) // Something distinctive, is sometimes useful to identify this externally.
	}
}
