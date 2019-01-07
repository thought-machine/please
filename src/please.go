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
	"sync"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cache"
	"github.com/thought-machine/please/src/clean"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/export"
	"github.com/thought-machine/please/src/follow"
	"github.com/thought-machine/please/src/gc"
	"github.com/thought-machine/please/src/hashes"
	"github.com/thought-machine/please/src/help"
	"github.com/thought-machine/please/src/ide/intellij"
	"github.com/thought-machine/please/src/metrics"
	"github.com/thought-machine/please/src/output"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/plz"
	"github.com/thought-machine/please/src/query"
	"github.com/thought-machine/please/src/run"
	"github.com/thought-machine/please/src/test"
	"github.com/thought-machine/please/src/tool"
	"github.com/thought-machine/please/src/update"
	"github.com/thought-machine/please/src/utils"
	"github.com/thought-machine/please/src/watch"
	"github.com/thought-machine/please/src/worker"
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
		Verbosity         cli.Verbosity `short:"v" long:"verbosity" description:"Verbosity of output (error, warning, notice, info, debug)" default:"warning"`
		LogFile           cli.Filepath  `long:"log_file" description:"File to echo full logging output to" default:"plz-out/log/build.log"`
		LogFileLevel      cli.Verbosity `long:"log_file_level" description:"Log level for file output" default:"debug"`
		InteractiveOutput bool          `long:"interactive_output" description:"Show interactive output in a terminal"`
		PlainOutput       bool          `short:"p" long:"plain_output" description:"Don't show interactive output."`
		Colour            bool          `long:"colour" description:"Forces coloured output from logging & other shell output."`
		NoColour          bool          `long:"nocolour" description:"Forces colourless output from logging & other shell output."`
		TraceFile         cli.Filepath  `long:"trace_file" description:"File to write Chrome tracing output into"`
		ShowAllOutput     bool          `long:"show_all_output" description:"Show all output live from all commands. Implies --plain_output."`
		CompletionScript  bool          `long:"completion_script" description:"Prints the bash / zsh completion script to stdout"`
	} `group:"Options controlling output & logging"`

	FeatureFlags struct {
		NoUpdate           bool `long:"noupdate" description:"Disable Please attempting to auto-update itself."`
		NoCache            bool `long:"nocache" description:"Disable caches (NB. not incrementality)"`
		NoHashVerification bool `long:"nohash_verification" description:"Hash verification errors are nonfatal."`
		NoLock             bool `long:"nolock" description:"Don't attempt to lock the repo exclusively. Use with care."`
		KeepWorkdirs       bool `long:"keep_workdirs" description:"Don't clean directories in plz-out/tmp after successfully building targets."`
	} `group:"Options that enable / disable certain features"`

	HelpFlags struct {
		Help    bool `short:"h" long:"help" description:"Show this help message"`
		Version bool `long:"version" description:"Print the version of Please"`
	} `group:"Help Options"`

	Profile          string `long:"profile_file" hidden:"true" description:"Write profiling output to this file"`
	MemProfile       string `long:"mem_profile_file" hidden:"true" description:"Write a memory profile to this file"`
	ProfilePort      int    `long:"profile_port" hidden:"true" description:"Serve profiling info on this port."`
	ParsePackageOnly bool   `description:"Parses a single package only. All that's necessary for some commands." no-flag:"true"`
	Complete         string `long:"complete" hidden:"true" env:"PLZ_COMPLETE" description:"Provide completion options for this build target."`

	Build struct {
		Prepare    bool     `long:"prepare" description:"Prepare build directory for these targets but don't build them."`
		Shell      bool     `long:"shell" description:"Like --prepare, but opens a shell in the build directory with the appropriate environment variables."`
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
		NumRuns         int          `long:"num_runs" short:"n" default:"1" description:"Number of times to run each test target."`
		TestResultsFile cli.Filepath `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		SurefireDir     cli.Filepath `long:"surefire_dir" default:"plz-out/surefire-reports" description:"Directory to copy XML test results to."`
		ShowOutput      bool         `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		Debug           bool         `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest, C and C++). Implies -c dbg unless otherwise set."`
		Failed          bool         `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Detailed        bool         `long:"detailed" description:"Prints more detailed output after tests."`
		Shell           bool         `long:"shell" description:"Opens a shell in the test directory with the appropriate environment variables."`
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
		NumRuns             int          `short:"n" long:"num_runs" default:"1" description:"Number of times to run each test target."`
		IncludeAllFiles     bool         `short:"a" long:"include_all_files" description:"Include all dependent files in coverage (default is just those from relevant packages)"`
		IncludeFile         []string     `long:"include_file" description:"Filenames to filter coverage display to"`
		TestResultsFile     cli.Filepath `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		SurefireDir         cli.Filepath `long:"surefire_dir" default:"plz-out/surefire-reports" description:"Directory to copy XML test results to."`
		CoverageResultsFile cli.Filepath `long:"coverage_results_file" default:"plz-out/log/coverage.json" description:"File to write combined coverage results to."`
		CoverageXMLReport   cli.Filepath `long:"coverage_xml_report" default:"plz-out/log/coverage.xml" description:"XML File to write combined coverage results to."`
		ShowOutput          bool         `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		Debug               bool         `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest, C and C++). Implies -c dbg unless otherwise set."`
		Failed              bool         `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Detailed            bool         `long:"detailed" description:"Prints more detailed output after tests."`
		Shell               bool         `long:"shell" description:"Opens a shell in the test directory with the appropriate environment variables."`
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
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Labels to export."`
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
			URL cli.URL `positional-arg-name:"URL" required:"yes" description:"URL of remote server to connect to, e.g. 10.23.0.5:7777"`
		} `positional-args:"true" required:"yes"`
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
			Level  int  `long:"level" default:"-1" description:"levels of the dependencies to retrieve."`
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
		Changes struct {
			Since           string `short:"s" long:"since" default:"origin/master" description:"Revision to compare against"`
			CheckoutCommand string `long:"checkout_command" default:"git checkout %s" description:"Command to run to check out the before/after revisions."`
			CurrentCommand  string `long:"current_revision_command" default:"git rev-parse --abbrev-ref HEAD" description:"Command to run to get the current revision (which will be checked out again at the end)"`
			Args            struct {
				Files cli.StdinStrings `positional-arg-name:"files" description:"Files to consider changed"`
			} `positional-args:"true"`
		} `command:"changes" description:"Calculates the difference between two different states of the build graph"`
		Roots struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true"`
		} `command:"roots" description:"Show build labels with no dependents in the given list, from the list."`
		Filter struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to filter"`
			} `positional-args:"true"`
		} `command:"filter" description:"Filter the given set of targets according to some rules"`
		Changed struct {
			Since            string `long:"since" description:"Calculate changes since this tree-ish/scm ref (defaults to current HEAD/tip)."`
			DiffSpec         string `long:"diffspec" description:"Calculate changes contained within given scm spec (commit range/sha/ref/etc)."`
			IncludeDependees string `long:"include-dependees" default:"none" choice:"none" choice:"direct" choice:"transitive" description:"Include direct or transitive dependees of changed targets."`
		} `command:"changed" description:"Show changed targets since some diffspec."`
	} `command:"query" description:"Queries information about the build graph"`

	Ide struct {
		IntelliJ struct {
			Args struct {
				Labels []core.BuildLabel `positional-arg-name:"labels" description:"Targets to include."`
			} `positional-args:"true"`
		} `command:"intellij" description:"Export intellij structure for the given targets and their dependencies."`
	} `command:"ide" description:"IDE Support and generation."`
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
		success, _ := doTest(targets, opts.Test.SurefireDir, opts.Test.TestResultsFile)
		return success || opts.Test.FailingTestsOk
	},
	"cover": func() bool {
		if opts.BuildFlags.Config != "" {
			log.Warning("Build config overridden; coverage may not be available for some languages")
		} else {
			opts.BuildFlags.Config = "cover"
		}
		targets := testTargets(opts.Cover.Args.Target, opts.Cover.Args.Args, opts.Cover.Failed, opts.Cover.TestResultsFile)
		os.RemoveAll(string(opts.Cover.CoverageResultsFile))
		success, state := doTest(targets, opts.Cover.SurefireDir, opts.Cover.TestResultsFile)
		test.AddOriginalTargetsToCoverage(state, opts.Cover.IncludeAllFiles)
		test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension)

		test.WriteCoverageToFileOrDie(state.Coverage, string(opts.Cover.CoverageResultsFile))
		test.WriteXMLCoverageToFileOrDie(targets, state.Coverage, string(opts.Cover.CoverageXMLReport))

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
	"init": func() bool {
		utils.InitConfig(string(opts.Init.Dir), opts.Init.BazelCompatibility)
		return true
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
		state := core.NewBuildState(1, nil, int(opts.OutputFlags.Verbosity), config)
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
			query.Deps(state, state.ExpandOriginalTargets(), opts.Query.Deps.Unique, opts.Query.Deps.Level)
		})
	},
	"reverseDeps": func() bool {
		return runQuery(false, opts.Query.ReverseDeps.Args.Targets, func(state *core.BuildState) {
			query.ReverseDeps(state, state.ExpandOriginalTargets())
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
			query.AffectedTargets(state, files.Get(), opts.BuildFlags.Include, opts.BuildFlags.Exclude, opts.Query.AffectedTargets.Tests, !opts.Query.AffectedTargets.Intransitive)
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
		if success, state := Please(parseLabels, config, false, false); success {
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
		success, state := Please(opts.Query.Rules.Args.Targets, config, len(opts.Query.Rules.Args.Targets) > 0, false)
		if success {
			parse.PrintRuleArgs(state, state.ExpandOriginalTargets())
		}
		return success
	},
	"changed": func() bool {
		success, state := runBuild(core.WholeGraph, false, false)
		if !success {
			return false
		}
		for _, label := range query.ChangedLabels(
			state,
			query.ChangedRequest{
				Since:            opts.Query.Changed.Since,
				DiffSpec:         opts.Query.Changed.DiffSpec,
				IncludeDependees: opts.Query.Changed.IncludeDependees,
			}) {
			fmt.Printf("%s\n", label)
		}
		return true
	},
	"changes": func() bool {
		// Temporarily set this flag on to avoid fatal errors from the first parse.
		keepGoing := opts.BuildFlags.KeepGoing
		opts.BuildFlags.KeepGoing = true
		original := query.MustGetRevision(opts.Query.Changes.CurrentCommand)
		files := opts.Query.Changes.Args.Files.Get()
		query.MustCheckout(opts.Query.Changes.Since, opts.Query.Changes.CheckoutCommand)
		_, before := runBuild(core.WholeGraph, false, false)
		opts.BuildFlags.KeepGoing = keepGoing
		// N.B. Ignore failure here; if we can't parse the graph before then it will suffice to
		//      assume that anything we don't know about has changed.
		query.MustCheckout(original, opts.Query.Changes.CheckoutCommand)
		success, after := runBuild(core.WholeGraph, false, false)
		if !success {
			return false
		}
		for _, target := range query.DiffGraphs(before, after, files) {
			fmt.Printf("%s\n", target)
		}
		return true
	},
	"roots": func() bool {
		return runQuery(true, opts.Query.Roots.Args.Targets, func(state *core.BuildState) {
			query.Roots(state.Graph, opts.Query.Roots.Args.Targets)
		})
	},
	"watch": func() bool {
		// Don't ask it to test now since we don't know if any of them are tests yet.
		success, state := runBuild(opts.Watch.Args.Targets, true, false)
		state.NeedRun = opts.Watch.Run
		watch.Watch(state, state.ExpandOriginalTargets(), runPlease)
		return success
	},
	"filter": func() bool {
		return runQuery(false, opts.Query.Filter.Args.Targets, func(state *core.BuildState) {
			query.Filter(state, state.ExpandOriginalTargets())
		})
	},
	"intellij": func() bool {
		success, state := runBuild(opts.Ide.IntelliJ.Args.Labels, false, false)
		if success {
			intellij.ExportIntellijStructure(state.Config, state.Graph, state.ExpandOriginalLabels())
		}
		return success
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

func setWatchedTarget(state *core.BuildState, labels core.BuildLabels) string {
	if opts.Watch.Run {
		opts.Run.Parallel.PositionalArgs.Targets = labels
		return "parallel"
	}

	for i, label := range labels {
		if state.Graph.TargetOrDie(label).IsTest {
			if i == 0 {
				opts.Test.Args.Target = label
			}
			opts.Test.Args.Args = append(opts.Test.Args.Args, label.String())
		}

		if i == len(labels)-1 && opts.Test.Args.Target.Name != "" {
			return "test"
		}
	}
	opts.Build.Args.Targets = labels
	return "build"
}

func doTest(targets []core.BuildLabel, surefireDir cli.Filepath, resultsFile cli.Filepath) (bool, *core.BuildState) {
	os.RemoveAll(string(surefireDir))
	os.RemoveAll(string(resultsFile))
	os.MkdirAll(string(surefireDir), core.DirPermissions)
	success, state := runBuild(targets, true, true)
	test.CopySurefireXmlFilesToDir(state, string(surefireDir))
	test.WriteResultsToFileOrDie(state.Graph, string(resultsFile))
	return success, state
}

// prettyOutputs determines from input flags whether we should show 'pretty' output (ie. interactive).
func prettyOutput(interactiveOutput bool, plainOutput bool, verbosity cli.Verbosity) bool {
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
func Please(targets []core.BuildLabel, config *core.Configuration, shouldBuild, shouldTest bool) (bool, *core.BuildState) {
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
	state := core.NewBuildState(config.Please.NumThreads, nil, int(opts.OutputFlags.Verbosity), config)
	state.VerifyHashes = !opts.FeatureFlags.NoHashVerification
	state.NumTestRuns = utils.Max(opts.Test.NumRuns, opts.Cover.NumRuns)  // Only one of these can be passed
	state.TestArgs = append(opts.Test.Args.Args, opts.Cover.Args.Args...) // Similarly here.
	state.NeedCoverage = !opts.Cover.Args.Target.IsEmpty()
	state.NeedBuild = shouldBuild
	state.NeedTests = shouldTest
	state.NeedRun = !opts.Run.Args.Target.IsEmpty() || len(opts.Run.Parallel.PositionalArgs.Targets) > 0 || len(opts.Run.Sequential.PositionalArgs.Targets) > 0
	state.NeedHashesOnly = len(opts.Hash.Args.Targets) > 0
	state.PrepareOnly = opts.Build.Prepare || opts.Build.Shell
	state.PrepareShell = opts.Build.Shell || opts.Test.Shell || opts.Cover.Shell
	state.Watch = len(opts.Watch.Args.Targets) > 0
	state.CleanWorkdirs = !opts.FeatureFlags.KeepWorkdirs
	state.ForceRebuild = len(opts.Rebuild.Args.Targets) > 0
	state.ShowTestOutput = opts.Test.ShowOutput || opts.Cover.ShowOutput
	state.DebugTests = debugTests
	state.ShowAllOutput = opts.OutputFlags.ShowAllOutput
	state.ParsePackageOnly = opts.ParsePackageOnly
	state.SetIncludeAndExclude(opts.BuildFlags.Include, opts.BuildFlags.Exclude)

	if state.DebugTests && len(targets) != 1 {
		log.Fatalf("-d/--debug flag can only be used with a single test target")
	}

	runPlease(state, targets)
	return state.Success, state
}

func runPlease(state *core.BuildState, targets []core.BuildLabel) {
	// Acquire the lock before we start building
	if (state.NeedBuild || state.NeedTests) && !opts.FeatureFlags.NoLock {
		core.AcquireRepoLock()
		defer core.ReleaseRepoLock()
	}

	detailedTests := state.NeedTests && (opts.Test.Detailed || opts.Cover.Detailed ||
		(len(targets) == 1 && !targets[0].IsAllTargets() &&
			!targets[0].IsAllSubpackages() && targets[0] != core.BuildLabelStdin))
	pretty := prettyOutput(opts.OutputFlags.InteractiveOutput, opts.OutputFlags.PlainOutput, opts.OutputFlags.Verbosity) && state.NeedBuild
	state.Cache = newCache(config)

	// Run the display
	state.Results() // important this is called now, don't ask...
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		output.MonitorState(state, config.Please.NumThreads, !pretty, opts.BuildFlags.KeepGoing, detailedTests, string(opts.OutputFlags.TraceFile))
		wg.Done()
	}()

	plz.Run(targets, state, config, opts.BuildFlags.Arch)
	wg.Wait()
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

	config, err := core.ReadDefaultConfigFiles(opts.BuildFlags.Profile)
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
	return Please(targets, config, shouldBuild, shouldTest)
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
	cli.InitLogging(cli.MinVerbosity) // Ensure this is quiet
	opts.FeatureFlags.NoUpdate = true // Ensure we don't try to update
	if len(items) > 0 && strings.HasPrefix(items[0].Item, "//") {
		// Don't muck around with the config if we're predicting build labels.
		cli.PrintCompletions(items)
	} else if config := readConfigAndSetRoot(false); config.AttachAliasFlags(parser) {
		// Run again without this registered as a completion handler
		parser.CompletionHandler = nil
		parser.ParseArgs(os.Args[1:])
	} else {
		cli.PrintCompletions(items)
	}
	// Regardless of what happened, always exit with 0 at this point.
	os.Exit(0)
}

func initBuild(args []string) string {
	if _, present := os.LookupEnv("GO_FLAGS_COMPLETION"); present {
		cli.InitLogging(cli.MinVerbosity)
	}
	parser, extraArgs, flagsErr := cli.ParseFlags("Please", &opts, args, flags.PassDoubleDash, handleCompletions)
	// Note that we must leave flagsErr for later, because it may be affected by aliases.
	if opts.HelpFlags.Version {
		fmt.Printf("Please version %s\n", core.PleaseVersion)
		os.Exit(0) // Ignore other flags if --version was passed.
	} else if opts.HelpFlags.Help {
		// Attempt to read config files to produce help for aliases.
		cli.InitLogging(cli.MinVerbosity)
		parser.WriteHelp(os.Stderr)
		if core.FindRepoRoot() {
			if config, err := core.ReadDefaultConfigFiles(""); err == nil {
				config.PrintAliases(os.Stderr)
			}
		}
		os.Exit(0)
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

	command := cli.ActiveCommand(parser.Command)
	if opts.Complete != "" {
		// Completion via PLZ_COMPLETE env var sidesteps other commands
		opts.Query.Completions.Cmd = command
		opts.Query.Completions.Args.Fragments = []string{opts.Complete}
		command = "completions"
	} else if command == "help" || command == "follow" || command == "init" {
		// These commands don't use a config file, allowing them to be run outside a repo.
		if flagsErr != nil { // This error otherwise doesn't get checked until later.
			cli.ParseFlagsFromArgsOrDie("Please", &opts, os.Args)
		}
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
	if parser.Command.Active != nil && parser.Command.Active.Name == "query" {
		// Query commands don't need either of these set.
		opts.OutputFlags.PlainOutput = true
		config.Cache.DirClean = false
	}

	// Now we've read the config file, we may need to re-run the parser; the aliases in the config
	// can affect how we parse otherwise illegal flag combinations.
	if (flagsErr != nil || len(extraArgs) > 0) && command != "completions" {
		args := config.UpdateArgsWithAliases(os.Args)
		command = cli.ParseFlagsFromArgsOrDie("Please", &opts, args)
	}

	if opts.ProfilePort != 0 {
		go func() {
			log.Warning("%s", http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", opts.ProfilePort), nil))
		}()
	}
	return command
}

func execute(command string) bool {
	if opts.Profile != "" {
		f, err := os.Create(opts.Profile)
		if err != nil {
			log.Fatalf("Failed to open profile file: %s", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start profiler: %s", err)
		}
		defer f.Close()
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

	success := buildFunctions[command]()

	metrics.Stop()
	worker.StopAll()

	return success
}

func main() {
	command := initBuild(os.Args)
	success := execute(command)

	if !success {
		os.Exit(7) // Something distinctive, is sometimes useful to identify this externally.
	}
}
