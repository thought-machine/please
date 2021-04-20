package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"

	"github.com/thought-machine/go-flags"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/assets"
	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cache"
	"github.com/thought-machine/please/src/clean"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/export"
	"github.com/thought-machine/please/src/format"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/gc"
	"github.com/thought-machine/please/src/generate"
	"github.com/thought-machine/please/src/hashes"
	"github.com/thought-machine/please/src/help"
	"github.com/thought-machine/please/src/output"
	"github.com/thought-machine/please/src/plz"
	"github.com/thought-machine/please/src/plzinit"
	"github.com/thought-machine/please/src/query"
	"github.com/thought-machine/please/src/run"
	"github.com/thought-machine/please/src/scm"
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
		Config     string              `short:"c" long:"config" env:"PLZ_BUILD_CONFIG" description:"Build config to use. Defaults to opt."`
		Arch       cli.Arch            `short:"a" long:"arch" description:"Architecture to compile for."`
		RepoRoot   cli.Filepath        `short:"r" long:"repo_root" description:"Root of repository to build."`
		NumThreads int                 `short:"n" long:"num_threads" description:"Number of concurrent build operations. Default is number of CPUs + 2."`
		Include    []string            `short:"i" long:"include" description:"Label of targets to include in automatic detection."`
		Exclude    []string            `short:"e" long:"exclude" description:"Label of targets to exclude from automatic detection."`
		Option     ConfigOverrides     `short:"o" long:"override" env:"PLZ_OVERRIDES" env-delim:";" description:"Options to override from .plzconfig (e.g. -o please.selfupdate:false)"`
		Profile    core.ConfigProfiles `long:"profile" env:"PLZ_CONFIG_PROFILE" description:"Configuration profile to load; e.g. --profile=dev will load .plzconfig.dev if it exists."`
		PreTargets []core.BuildLabel   `long:"pre" hidden:"true" description:"Targets to build before the other command-line ones. Sometimes useful to debug targets generated as part of a post-build function."`
	} `group:"Options controlling what to build & how to build it"`

	OutputFlags struct {
		Verbosity         cli.Verbosity `short:"v" long:"verbosity" description:"Verbosity of output (error, warning, notice, info, debug)" default:"warning"`
		LogFile           cli.Filepath  `long:"log_file" description:"File to echo full logging output to" default:"plz-out/log/build.log"`
		LogFileLevel      cli.Verbosity `long:"log_file_level" description:"Log level for file output" default:"debug"`
		LogAppend         bool          `long:"log_append" description:"Append log to existing file instead of overwriting its content"`
		InteractiveOutput bool          `long:"interactive_output" description:"Show interactive output in a terminal"`
		PlainOutput       bool          `short:"p" long:"plain_output" description:"Don't show interactive output."`
		Colour            bool          `long:"colour" description:"Forces coloured output from logging & other shell output."`
		NoColour          bool          `long:"nocolour" description:"Forces colourless output from logging & other shell output."`
		TraceFile         cli.Filepath  `long:"trace_file" description:"File to write Chrome tracing output into"`
		ShowAllOutput     bool          `long:"show_all_output" description:"Show all output live from all commands. Implies --plain_output."`
		CompletionScript  bool          `long:"completion_script" description:"Prints the bash / zsh completion script to stdout"`
	} `group:"Options controlling output & logging"`

	FeatureFlags struct {
		NoUpdate           bool    `long:"noupdate" description:"Disable Please attempting to auto-update itself."`
		NoHashVerification bool    `long:"nohash_verification" description:"Hash verification errors are nonfatal."`
		NoLock             bool    `long:"nolock" description:"Don't attempt to lock the repo exclusively. Use with care."`
		KeepWorkdirs       bool    `long:"keep_workdirs" description:"Don't clean directories in plz-out/tmp after successfully building targets."`
		HTTPProxy          cli.URL `long:"http_proxy" env:"HTTP_PROXY" description:"HTTP proxy to use for downloads"`
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
		Prepare    bool `long:"prepare" description:"Prepare build directory for these targets but don't build them."`
		Shell      bool `long:"shell" description:"Like --prepare, but opens a shell in the build directory with the appropriate environment variables."`
		Rebuild    bool `long:"rebuild" description:"To force the optimisation and rebuild one or more targets."`
		NoDownload bool `long:"nodownload" hidden:"true" description:"Don't download outputs after building. Only applies when using remote build execution."`
		Download   bool `long:"download" hidden:"true" description:"Force download of all outputs regardless of original target spec. Only applies when using remote build execution."`
		Args       struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to build"`
		} `positional-args:"true" required:"true"`
	} `command:"build" description:"Builds one or more targets"`

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
		Rerun           bool         `long:"rerun" description:"Rerun the test even if the hash hasn't changed."`
		Sequentially    bool         `long:"sequentially" description:"Whether to run multiple runs of the same test sequentially"`
		TestResultsFile cli.Filepath `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		SurefireDir     cli.Filepath `long:"surefire_dir" default:"plz-out/surefire-reports" description:"Directory to copy XML test results to."`
		ShowOutput      bool         `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		Debug           bool         `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest, C and C++). Implies -c dbg unless otherwise set."`
		Failed          bool         `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Detailed        bool         `long:"detailed" description:"Prints more detailed output after tests."`
		Shell           bool         `long:"shell" description:"Opens a shell in the test directory with the appropriate environment variables."`
		StreamResults   bool         `long:"stream_results" description:"Prints test results on stdout as they are run."`
		// Slightly awkward since we can specify a single test with arguments or multiple test targets.
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors"`
		} `positional-args:"true"`
	} `command:"test" description:"Builds and tests one or more targets"`

	Cover struct {
		active              bool          `no-flag:"true"`
		FailingTestsOk      bool          `long:"failing_tests_ok" hidden:"true" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)"`
		NoCoverageReport    bool          `long:"nocoverage_report" description:"Suppress the per-file coverage report displayed in the shell"`
		LineCoverageReport  bool          `short:"l" long:"line_coverage_report" description:" Show a line-by-line coverage report for all affected files."`
		NumRuns             int           `short:"n" long:"num_runs" default:"1" description:"Number of times to run each test target."`
		Rerun               bool          `long:"rerun" description:"Rerun the test even if the hash hasn't changed."`
		Sequentially        bool          `long:"sequentially" description:"Whether to run multiple runs of the same test sequentially"`
		IncludeAllFiles     bool          `short:"a" long:"include_all_files" description:"Include all dependent files in coverage (default is just those from relevant packages)"`
		IncludeFile         cli.Filepaths `long:"include_file" description:"Filenames to filter coverage display to. Supports shell pattern matching e.g. file/path/*."`
		TestResultsFile     cli.Filepath  `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		SurefireDir         cli.Filepath  `long:"surefire_dir" default:"plz-out/surefire-reports" description:"Directory to copy XML test results to."`
		CoverageResultsFile cli.Filepath  `long:"coverage_results_file" default:"plz-out/log/coverage.json" description:"File to write combined coverage results to."`
		CoverageXMLReport   cli.Filepath  `long:"coverage_xml_report" default:"plz-out/log/coverage.xml" description:"XML File to write combined coverage results to."`
		Incremental         bool          `short:"i" long:"incremental" description:"Calculates summary statistics for incremental coverage, i.e. stats for just the lines currently modified."`
		ShowOutput          bool          `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		Debug               bool          `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest, C and C++). Implies -c dbg unless otherwise set."`
		Failed              bool          `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Detailed            bool          `long:"detailed" description:"Prints more detailed output after tests."`
		Shell               bool          `long:"shell" description:"Opens a shell in the test directory with the appropriate environment variables."`
		StreamResults       bool          `long:"stream_results" description:"Prints test results on stdout as they are run."`
		Args                struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test" group:"one test"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments or test selectors" group:"one test"`
		} `positional-args:"true"`
	} `command:"cover" description:"Builds and tests one or more targets, and calculates coverage."`

	Run struct {
		Env        bool   `long:"env" description:"Overrides environment variables (e.g. PATH) in the new process."`
		Rebuild    bool   `long:"rebuild" description:"To force the optimisation and rebuild one or more targets."`
		InWD       bool   `long:"in_wd" description:"When running locally, stay in the original working directory."`
		InTempDir  bool   `long:"in_tmp_dir" description:"Runs in a temp directory, setting env variables and copying in runtime data similar to tests."`
		EntryPoint string `long:"entry_point" short:"e" description:"The entry point of the target to use." default:""`
		Parallel   struct {
			NumTasks       int  `short:"n" long:"num_tasks" default:"10" description:"Maximum number of subtasks to run in parallel"`
			Quiet          bool `short:"q" long:"quiet" description:"Suppress output from successful subprocesses."`
			PositionalArgs struct {
				Targets []core.AnnotatedOutputLabel `positional-arg-name:"target" description:"Targets to run"`
			} `positional-args:"true" required:"true"`
			Args   cli.Filepaths `short:"a" long:"arg" description:"Arguments to pass to the called processes."`
			Detach bool          `long:"detach" description:"Detach from the parent process when all children have spawned"`
		} `command:"parallel" description:"Runs a sequence of targets in parallel"`
		Sequential struct {
			Quiet          bool `short:"q" long:"quiet" description:"Suppress output from successful subprocesses."`
			PositionalArgs struct {
				Targets []core.AnnotatedOutputLabel `positional-arg-name:"target" description:"Targets to run"`
			} `positional-args:"true" required:"true"`
			Args cli.Filepaths `short:"a" long:"arg" description:"Arguments to pass to the called processes."`
		} `command:"sequential" description:"Runs a sequence of targets sequentially."`
		Args struct {
			Target core.AnnotatedOutputLabel `positional-arg-name:"target" required:"true" description:"Target to run"`
			Args   cli.Filepaths             `positional-arg-name:"arguments" description:"Arguments to pass to target when running (to pass flags to the target, put -- before them)"`
		} `positional-args:"true"`
		Remote bool `long:"remote" description:"Send targets to be executed remotely."`
	} `command:"run" subcommands-optional:"true" description:"Builds and runs a single target"`

	Clean struct {
		NoBackground bool     `long:"nobackground" short:"f" description:"Don't fork & detach until clean is finished."`
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
		Force            bool        `long:"force" description:"Forces a re-download of the new version."`
		NoVerify         bool        `long:"noverify" description:"Skips signature verification of downloaded version"`
		Latest           bool        `long:"latest" description:"Update to latest available version (overrides config)."`
		LatestPrerelease bool        `long:"latest_prerelease" description:"Update to latest available prerelease version (overrides config)."`
		Version          cli.Version `long:"version" description:"Updates to a particular version (overrides config)."`
	} `command:"update" description:"Checks for an update and updates if needed."`

	Op struct {
	} `command:"op" description:"Re-runs previous command."`

	Init struct {
		Dir                cli.Filepath `long:"dir" description:"Directory to create config in" default:"."`
		BazelCompatibility bool         `long:"bazel_compat" description:"Initialises config for Bazel compatibility mode."`
		NoPrompt           bool         `long:"no_prompt" description:"Don't interactively prompt for optional config'"`
		Config             struct {
			User  bool `short:"u" long:"user" description:"Modifies the user-level config file"`
			Local bool `short:"l" long:"local" description:"Modifies the local config file (.plzconfig.local)"`
			Args  struct {
				Options ConfigOverrides `positional-arg-name:"config" required:"true" description:"Attributes to set"`
			} `positional-args:"true" required:"true"`
		} `command:"config" description:"Initialises specific attributes of config files"`
		Pleasings struct {
			Revision  string `short:"r" long:"revision" description:"The revision to pin the pleasings repo to. This can be a branch, commit, tag, or other git reference."`
			Location  string `short:"l" long:"location" description:"The location of the build file to write the subrepo rule to" default:"BUILD"`
			PrintOnly bool   `long:"print" description:"Print the rule to standard out instead of writing it to a file"`
		} `command:"pleasings" description:"Initialises the pleasings repo"`
		Pleasew struct {
		} `command:"pleasew" description:"Initialises the pleasew wrapper script"`
	} `command:"init" subcommands-optional:"true" description:"Initialises a .plzconfig file in the current directory"`

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

	Format struct {
		Quiet bool `long:"quiet" short:"q" description:"Don't print corrections to stdout, simply exit with a code indicating success / failure (for linting etc)."`
		Write bool `long:"write" short:"w" description:"Rewrite files after update"`
		Args  struct {
			Files cli.Filepaths `positional-arg-name:"files" description:"BUILD files to reformat"`
		} `positional-args:"true"`
	} `command:"format" alias:"fmt" description:"Autoformats BUILD files"`

	Help struct {
		Args struct {
			Topic help.Topic `positional-arg-name:"topic" description:"Topic to display help on"`
		} `positional-args:"true"`
	} `command:"help" alias:"halp" description:"Displays help about various parts of plz or its build rules"`

	Tool struct {
		Args struct {
			Tool tool.Tool     `positional-arg-name:"tool" description:"Tool to invoke (jarcat, lint, etc)"`
			Args cli.Filepaths `positional-arg-name:"arguments" description:"Arguments to pass to the tool"`
		} `positional-args:"true"`
	} `command:"tool" hidden:"true" description:"Invoke one of Please's sub-tools"`

	Query struct {
		Deps struct {
			Hidden bool `long:"hidden" short:"h" description:"Output internal / hidden dependencies too"`
			Level  int  `long:"level" default:"-1" description:"Levels of the dependencies to retrieve."`
			Unique bool `long:"unique" hidden:"true" description:"Has no effect, only exists for compatibility."`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"deps" description:"Queries the dependencies of a target."`
		ReverseDeps struct {
			Level  int  `long:"level" default:"1" description:"Levels of the dependencies to retrieve (-1 for unlimited)."`
			Hidden bool `long:"hidden" short:"h" description:"Output internal / hidden dependencies too"`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"revdeps" alias:"reverseDeps" description:"Queries all the reverse dependencies of a target."`
		SomePath struct {
			Args struct {
				Target1 core.BuildLabel `positional-arg-name:"target1" description:"First build target" required:"true"`
				Target2 core.BuildLabel `positional-arg-name:"target2" description:"Second build target" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"somepath" description:"Queries for a dependency path between two targets in the build graph"`
		AllTargets struct {
			Hidden bool `long:"hidden" description:"Show hidden targets as well"`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query"`
			} `positional-args:"true"`
		} `command:"alltargets" description:"Lists all targets in the graph"`
		Print struct {
			Fields []string `short:"f" long:"field" description:"Individual fields to print of the target"`
			Labels []string `short:"l" long:"label" description:"Prints all labels with the given prefix (with the prefix stripped off). Overrides --field."`
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
				Targets []core.BuildLabel `hidden:"true" description:"deprecated, has no effect"`
			} `positional-args:"true"`
		} `command:"rules" description:"Prints built-in rules to stdout as JSON"`
		Changes struct {
			Since            string `short:"s" long:"since" default:"origin/master" description:"Revision to compare against"`
			IncludeDependees string `long:"include_dependees" default:"none" choice:"none" choice:"direct" choice:"transitive" description:"Deprecated: use level 1 for direct and -1 for transitive. Include direct or transitive dependees of changed targets."`
			Level            int    `long:"level" default:"-2" description:"Levels of the dependencies of changed targets (-1 for unlimited)." default-mask:"0"`
			Inexact          bool   `long:"inexact" description:"Calculate changes more quickly and without doing any SCM checkouts, but may miss some targets."`
			In               string `long:"in" description:"Calculate changes contained within given scm spec (commit range/sha/ref/etc). Implies --inexact."`
			Args             struct {
				Files cli.StdinStrings `positional-arg-name:"files" description:"Files to calculate changes for. Overrides flags relating to SCM operations."`
			} `positional-args:"true"`
		} `command:"changes" description:"Calculates the set of changed targets in regard to a set of modified files or SCM commits."`
		Roots struct {
			Hidden bool `long:"hidden" description:"Show hidden targets as well"`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to query" required:"true"`
			} `positional-args:"true"`
		} `command:"roots" description:"Show build labels with no dependents in the given list, from the list."`
		Filter struct {
			Hidden bool `long:"hidden" description:"Show hidden targets as well"`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to filter"`
			} `positional-args:"true"`
		} `command:"filter" description:"Filter the given set of targets according to some rules"`
	} `command:"query" description:"Queries information about the build graph"`
	Codegen struct {
		Gitignore string `long:"update_gitignore" description:"The gitignore file to write the generated sources to"`
		Args      struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to filter"`
		} `positional-args:"true"`
	} `command:"generate" description:"Builds all code generation targets in the repository and prints the generated files."`
}

// Definitions of what we do for each command.
// Functions are called after args are parsed and return true for success.
var buildFunctions = map[string]func() int{
	"build": func() int {
		success, state := runBuild(opts.Build.Args.Targets, true, false, false)
		return toExitCode(success, state)
	},
	"hash": func() int {
		success, state := runBuild(opts.Hash.Args.Targets, true, false, false)
		if success {
			if opts.Hash.Detailed {
				for _, target := range state.ExpandOriginalLabels() {
					build.PrintHashes(state, state.Graph.TargetOrDie(target))
				}
			}
			if opts.Hash.Update {
				hashes.RewriteHashes(state, state.ExpandOriginalLabels())
			}
		}
		return toExitCode(success, state)
	},
	"test": func() int {
		targets := testTargets(opts.Test.Args.Target, opts.Test.Args.Args, opts.Test.Failed, opts.Test.TestResultsFile)
		success, state := doTest(targets, opts.Test.SurefireDir, opts.Test.TestResultsFile)
		return toExitCode(success, state)
	},
	"cover": func() int {
		opts.Cover.active = true
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

		var stats *test.IncrementalStats
		if opts.Cover.Incremental {
			lines, err := scm.NewFallback(core.RepoRoot).ChangedLines()
			if err != nil {
				log.Fatalf("Failed to determine changes: %s", err)
			}
			stats = test.CalculateIncrementalStats(state, lines)
		}
		test.WriteCoverageToFileOrDie(state.Coverage, string(opts.Cover.CoverageResultsFile), stats)
		test.WriteXMLCoverageToFileOrDie(targets, state.Coverage, string(opts.Cover.CoverageXMLReport))

		if opts.Cover.LineCoverageReport {
			output.PrintLineCoverageReport(state, opts.Cover.IncludeFile.AsStrings())
		} else if !opts.Cover.NoCoverageReport && !opts.Cover.Shell {
			output.PrintCoverage(state, opts.Cover.IncludeFile.AsStrings())
		}
		if opts.Cover.Incremental {
			output.PrintIncrementalCoverage(stats)
		}
		return toExitCode(success, state)
	},
	"run": func() int {
		if success, state := runBuild([]core.BuildLabel{opts.Run.Args.Target.BuildLabel}, true, false, false); success {
			var dir string
			if opts.Run.InWD {
				dir = originalWorkingDirectory
			}

			if opts.Run.EntryPoint != "" {
				opts.Run.Args.Target.Annotation = opts.Run.EntryPoint
			}

			annotatedOutputLabels := state.ExpandOriginalMaybeAnnotatedLabels([]core.AnnotatedOutputLabel{opts.Run.Args.Target})
			if len(annotatedOutputLabels) != 1 {
				log.Fatalf("%v expanded to too many targets: %v", opts.Run.Args.Target, annotatedOutputLabels)
			}

			run.Run(state, annotatedOutputLabels[0], opts.Run.Args.Args.AsStrings(), opts.Run.Remote, opts.Run.Env, opts.Run.InTempDir, dir)
		}
		return 1 // We should never return from run.Run so if we make it here something's wrong.
	},
	"parallel": func() int {
		if success, state := runBuild(unannotateLabels(opts.Run.Parallel.PositionalArgs.Targets), true, false, false); success {
			var dir string
			if opts.Run.InWD {
				dir = originalWorkingDirectory
			}
			ls := state.ExpandOriginalMaybeAnnotatedLabels(opts.Run.Parallel.PositionalArgs.Targets)
			os.Exit(run.Parallel(context.Background(), state, ls, opts.Run.Parallel.Args.AsStrings(), opts.Run.Parallel.NumTasks, opts.Run.Parallel.Quiet, opts.Run.Remote, opts.Run.Env, opts.Run.Parallel.Detach, opts.Run.InTempDir, dir))
		}
		return 1
	},
	"sequential": func() int {
		if success, state := runBuild(unannotateLabels(opts.Run.Sequential.PositionalArgs.Targets), true, false, false); success {
			var dir string
			if opts.Run.InWD {
				dir = originalWorkingDirectory
			}

			ls := state.ExpandOriginalMaybeAnnotatedLabels(opts.Run.Sequential.PositionalArgs.Targets)
			os.Exit(run.Sequential(state, ls, opts.Run.Sequential.Args.AsStrings(), opts.Run.Sequential.Quiet, opts.Run.Remote, opts.Run.Env, opts.Run.InTempDir, dir))
		}
		return 1
	},
	"clean": func() int {
		config.Cache.DirClean = false // don't run the normal cleaner
		if len(opts.Clean.Args.Targets) == 0 && core.InitialPackage()[0].PackageName == "" {
			if len(opts.BuildFlags.Include) == 0 && len(opts.BuildFlags.Exclude) == 0 {
				// Clean everything, doesn't require parsing at all.
				state := core.NewBuildState(config)
				clean.Clean(config, cache.NewCache(state), !opts.Clean.NoBackground)
				return 0
			}
			opts.Clean.Args.Targets = core.WholeGraph
		}
		if success, state := runBuild(opts.Clean.Args.Targets, false, false, false); success {
			clean.Targets(state, state.ExpandOriginalLabels())
			return 0
		}
		return 1
	},
	"update": func() int {
		fmt.Printf("Up to date (version %s).\n", core.PleaseVersion)
		return 0 // We'd have died already if something was wrong.
	},
	"op": func() int {
		cmd := core.ReadLastOperationOrDie()
		log.Notice("OP PLZ: %s", strings.Join(cmd, " "))
		// Annoyingly we don't seem to have any access to execvp() which would be rather useful here...
		executable, err := os.Executable()
		if err == nil {
			err = syscall.Exec(executable, append([]string{executable}, cmd...), os.Environ())
		}
		log.Fatalf("SORRY OP: %s", err) // On success Exec never returns.
		return 1
	},
	"gc": func() int {
		success, state := runBuild(core.WholeGraph, false, false, true)
		if success {
			gc.GarbageCollect(state, opts.Gc.Args.Targets, state.ExpandLabels(state.Config.Gc.Keep), state.Config.Gc.Keep, state.Config.Gc.KeepLabel,
				opts.Gc.Conservative, opts.Gc.TargetsOnly, opts.Gc.SrcsOnly, opts.Gc.NoPrompt, opts.Gc.DryRun, opts.Gc.Git)
		}
		return toExitCode(success, state)
	},
	"format": func() int {
		if changed, err := format.Format(config, opts.Format.Args.Files.AsStrings(), opts.Format.Write, opts.Format.Quiet); err != nil {
			log.Fatalf("Failed to reformat files: %s", err)
		} else if changed && !opts.Format.Write {
			return 1
		}
		return 0
	},
	"init": func() int {
		plzinit.InitConfig(string(opts.Init.Dir), opts.Init.BazelCompatibility, opts.Init.NoPrompt)

		if opts.Init.NoPrompt {
			return 0
		}

		fmt.Println()
		fmt.Println("Pleasings are a collection of auxiliary build rules that support other languages and technologies not present in the core please distribution.")
		fmt.Println("For more information visit https://github.com/thought-machine/pleasings")
		fmt.Println()

		return 0
	},
	"config": func() int {
		if opts.Init.Config.User {
			plzinit.InitConfigFile(fs.ExpandHomePath(core.UserConfigFileName), opts.Init.Config.Args.Options)
		} else if opts.Init.Config.Local {
			plzinit.InitConfigFile(core.LocalConfigFileName, opts.Init.Config.Args.Options)
		} else {
			plzinit.InitConfigFile(core.ConfigFileName, opts.Init.Config.Args.Options)
		}
		return 0
	},
	"export": func() int {
		success, state := runBuild(opts.Export.Args.Targets, false, false, false)
		if success {
			export.ToDir(state, opts.Export.Output, state.ExpandOriginalLabels())
		}
		return toExitCode(success, state)
	},
	"outputs": func() int {
		success, state := runBuild(opts.Export.Outputs.Args.Targets, true, false, true)
		if success {
			export.Outputs(state, opts.Export.Output, state.ExpandOriginalLabels())
		}
		return toExitCode(success, state)
	},
	"help": func() int {
		return toExitCode(help.Help(string(opts.Help.Args.Topic)), nil)
	},
	"tool": func() int {
		tool.Run(config, opts.Tool.Args.Tool, opts.Tool.Args.Args.AsStrings())
		return 1 // If the function returns (which it shouldn't), something went wrong.
	},
	"deps": func() int {
		return runQuery(true, opts.Query.Deps.Args.Targets, func(state *core.BuildState) {
			query.Deps(state, state.ExpandOriginalLabels(), opts.Query.Deps.Hidden, opts.Query.Deps.Level)
		})
	},
	"revdeps": func() int {
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			query.ReverseDeps(state, state.ExpandLabels(utils.ReadStdinLabels(opts.Query.ReverseDeps.Args.Targets)), opts.Query.ReverseDeps.Level, opts.Query.ReverseDeps.Hidden)
		})
	},
	"somepath": func() int {
		a := utils.ReadStdinLabels([]core.BuildLabel{opts.Query.SomePath.Args.Target1})
		b := utils.ReadStdinLabels([]core.BuildLabel{opts.Query.SomePath.Args.Target2})
		return runQuery(true, append(a, b...), func(state *core.BuildState) {
			if err := query.SomePath(state.Graph, a, b); err != nil {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}
		})
	},
	"alltargets": func() int {
		return runQuery(true, opts.Query.AllTargets.Args.Targets, func(state *core.BuildState) {
			query.AllTargets(state.Graph, state.ExpandOriginalLabels(), opts.Query.AllTargets.Hidden)
		})
	},
	"print": func() int {
		return runQuery(false, opts.Query.Print.Args.Targets, func(state *core.BuildState) {
			query.Print(state.Graph, state.ExpandOriginalLabels(), opts.Query.Print.Fields, opts.Query.Print.Labels)
		})
	},
	"input": func() int {
		return runQuery(true, opts.Query.Input.Args.Targets, func(state *core.BuildState) {
			query.TargetInputs(state.Graph, state.ExpandOriginalLabels())
		})
	},
	"output": func() int {
		return runQuery(true, opts.Query.Output.Args.Targets, func(state *core.BuildState) {
			query.TargetOutputs(state.Graph, state.ExpandOriginalLabels())
		})
	},
	"completions": func() int {
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
			return 0
		}
		if len(fragments) == 0 || len(fragments) == 1 && strings.Trim(fragments[0], "/ ") == "" {
			os.Exit(0) // Don't do anything for empty completion, it's normally too slow.
		}
		labels, parseLabels, hidden := query.CompletionLabels(config, fragments, core.RepoRoot)
		if success, state := Please(parseLabels, config, false, false); success {
			binary := opts.Query.Completions.Cmd == "run"
			test := opts.Query.Completions.Cmd == "test" || opts.Query.Completions.Cmd == "cover"
			query.Completions(state.Graph, labels, binary, test, hidden)
			return 0
		}
		return 1
	},
	"graph": func() int {
		targets := opts.Query.Graph.Args.Targets
		return runQuery(true, targets, func(state *core.BuildState) {
			if len(opts.Query.Graph.Args.Targets) == 0 {
				targets = opts.Query.Graph.Args.Targets // It special-cases doing the full graph.
			}
			query.Graph(state, state.ExpandLabels(targets))
		})
	},
	"whatoutputs": func() int {
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			query.WhatOutputs(state.Graph, opts.Query.WhatOutputs.Args.Files.Get(), opts.Query.WhatOutputs.EchoFiles)
		})
	},
	"rules": func() int {
		help.PrintRuleArgs()
		return 0
	},
	"changes": func() int {
		// query changes always excludes 'manual' targets.
		opts.BuildFlags.Exclude = append(opts.BuildFlags.Exclude, "manual", "manual:"+core.OsArch)
		level := opts.Query.Changes.Level // -2 means unset -1 means all transitive
		transitive := opts.Query.Changes.IncludeDependees == "transitive"
		direct := opts.Query.Changes.IncludeDependees == "direct"
		if transitive || direct {
			log.Warning("include_dependees is deprecated. Please use level instead")
		}
		if (transitive || direct) && level != -2 {
			log.Warning("Both level and include_dependees are set. Using the value from level")
		}
		switch {
		// transitive subsumes direct so asses transitive first
		case transitive && (level == -2):
			level = -1
		case direct && (level == -2):
			level = 1
		case level == -2:
			level = 0
		}
		runInexact := func(files []string) int {
			return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
				for _, target := range query.Changes(state, files, level) {
					fmt.Println(target.String())
				}
			})
		}
		if len(opts.Query.Changes.Args.Files) > 0 {
			return runInexact(opts.Query.Changes.Args.Files.Get())
		}
		scm := scm.MustNew(core.RepoRoot)
		if opts.Query.Changes.In != "" {
			return runInexact(scm.ChangesIn(opts.Query.Changes.In, ""))
		} else if opts.Query.Changes.Inexact {
			return runInexact(scm.ChangedFiles(opts.Query.Changes.Since, true, ""))
		}
		original := scm.CurrentRevIdentifier()
		files := scm.ChangedFiles(opts.Query.Changes.Since, true, "")
		if err := scm.Checkout(opts.Query.Changes.Since); err != nil {
			log.Fatalf("%s", err)
		}
		readConfig(false)
		_, before := runBuild(core.WholeGraph, false, false, false)
		// N.B. Ignore failure here; if we can't parse the graph before then it will suffice to
		//      assume that anything we don't know about has changed.
		if err := scm.Checkout(original); err != nil {
			log.Fatalf("%s", err)
		}
		readConfig(false)
		success, after := runBuild(core.WholeGraph, false, false, false)
		if !success {
			return 1
		}
		for _, target := range query.DiffGraphs(before, after, files, level) {
			fmt.Println(target.String())
		}
		return 0
	},
	"roots": func() int {
		return runQuery(true, opts.Query.Roots.Args.Targets, func(state *core.BuildState) {
			query.Roots(state.Graph, state.ExpandOriginalLabels(), opts.Query.Roots.Hidden)
		})
	},
	"watch": func() int {
		// Don't ask it to test now since we don't know if any of them are tests yet.
		success, state := runBuild(opts.Watch.Args.Targets, true, false, false)
		state.NeedRun = opts.Watch.Run
		watch.Watch(state, state.ExpandOriginalLabels(), runPlease)
		return toExitCode(success, state)
	},
	"filter": func() int {
		return runQuery(false, opts.Query.Filter.Args.Targets, func(state *core.BuildState) {
			query.Filter(state, state.ExpandOriginalLabels(), opts.Query.Filter.Hidden)
		})
	},
	"pleasings": func() int {
		if err := plzinit.InitPleasings(opts.Init.Pleasings.Location, opts.Init.Pleasings.PrintOnly, opts.Init.Pleasings.Revision); err != nil {
			log.Fatalf("failed to write pleasings subrepo file: %v", err)
		}
		return 0
	},
	"pleasew": func() int {
		plzinit.InitWrapperScript()
		return 0
	},
	"generate": func() int {
		opts.BuildFlags.Include = append(opts.BuildFlags.Include, "codegen")

		if opts.Codegen.Gitignore != "" {
			pkg := filepath.Dir(opts.Codegen.Gitignore)
			if pkg == "." {
				pkg = ""
			}
			target := core.BuildLabel{
				PackageName: pkg,
				Name:        "...",
			}

			if len(opts.Codegen.Args.Targets) != 0 {
				log.Warning("You've provided targets, and a gitignore to update. Ignoring the provided targets and building %v", target)
			}

			opts.Codegen.Args.Targets = []core.BuildLabel{target}
		}

		if success, state := runBuild(opts.Codegen.Args.Targets, true, false, true); success {
			if opts.Codegen.Gitignore != "" {
				if !state.Config.Build.LinkGeneratedSources {
					log.Warning("You're updating a .gitignore with generated sources but Please isn't configured to link generated sources. See `plz help LinkGeneratedSources` for more information. ")
				}
				err := generate.UpdateGitignore(state.Graph, state.ExpandOriginalLabels(), opts.Codegen.Gitignore)
				if err != nil {
					log.Fatalf("failed to update gitignore: %v", err)
				}
			}
			return 0
		}
		return 1
	},
}

// ConfigOverrides are used to implement completion on the -o flag.
type ConfigOverrides map[string]string

// Complete implements the flags.Completer interface.
func (overrides ConfigOverrides) Complete(match string) []flags.Completion {
	return core.DefaultConfiguration().Completions(match)
}

// Used above as a convenience wrapper for query functions.
func runQuery(needFullParse bool, labels []core.BuildLabel, onSuccess func(state *core.BuildState)) int {
	if !needFullParse {
		opts.ParsePackageOnly = true
	}
	if len(labels) == 0 {
		labels = core.WholeGraph
	}
	if success, state := runBuild(labels, false, false, true); success {
		onSuccess(state)
		return 0
	}
	return 1
}

func doTest(targets []core.BuildLabel, surefireDir cli.Filepath, resultsFile cli.Filepath) (bool, *core.BuildState) {
	os.RemoveAll(string(surefireDir))
	os.RemoveAll(string(resultsFile))
	os.MkdirAll(string(surefireDir), core.DirPermissions)
	success, state := runBuild(targets, true, true, false)
	test.CopySurefireXMLFilesToDir(state, string(surefireDir))
	test.WriteResultsToFileOrDie(state.Graph, string(resultsFile), state.Config.Test.StoreTestOutputOnSuccess)
	return success, state
}

// prettyOutputs determines from input flags whether we should show 'pretty' output (ie. interactive).
func prettyOutput(interactiveOutput bool, plainOutput bool, verbosity cli.Verbosity) bool {
	if interactiveOutput && plainOutput {
		log.Fatal("Can't pass both --interactive_output and --plain_output")
	}
	return interactiveOutput || (!plainOutput && cli.StdErrIsATerminal && verbosity < 4)
}

// Please starts & runs the main build process through to its completion.
func Please(targets []core.BuildLabel, config *core.Configuration, shouldBuild, shouldTest bool) (bool, *core.BuildState) {
	if opts.BuildFlags.NumThreads > 0 {
		config.Please.NumThreads = opts.BuildFlags.NumThreads
		config.Parse.NumThreads = opts.BuildFlags.NumThreads
	}
	debugTests := opts.Test.Debug || opts.Cover.Debug
	if opts.BuildFlags.Config != "" {
		config.Build.Config = opts.BuildFlags.Config
	} else if debugTests {
		config.Build.Config = "dbg"
	}
	state := core.NewBuildState(config)
	state.VerifyHashes = !opts.FeatureFlags.NoHashVerification
	state.NumTestRuns = utils.Max(opts.Test.NumRuns, opts.Cover.NumRuns)       // Only one of these can be passed
	state.TestSequentially = opts.Test.Sequentially || opts.Cover.Sequentially // Similarly here.
	state.TestArgs = append(opts.Test.Args.Args, opts.Cover.Args.Args...)      // And here
	state.NeedCoverage = opts.Cover.active
	state.NeedBuild = shouldBuild
	state.NeedTests = shouldTest
	state.NeedRun = !opts.Run.Args.Target.IsEmpty() || len(opts.Run.Parallel.PositionalArgs.Targets) > 0 || len(opts.Run.Sequential.PositionalArgs.Targets) > 0
	state.NeedHashesOnly = len(opts.Hash.Args.Targets) > 0
	if opts.Build.Prepare {
		log.Warningf("--prepare has been deprecated in favour of --shell and will be removed in v17.")
	}
	state.PrepareOnly = opts.Build.Prepare || opts.Build.Shell
	state.PrepareShell = opts.Build.Shell || opts.Test.Shell || opts.Cover.Shell
	state.Watch = len(opts.Watch.Args.Targets) > 0
	state.CleanWorkdirs = !opts.FeatureFlags.KeepWorkdirs
	state.ForceRebuild = opts.Build.Rebuild || opts.Run.Rebuild
	state.ForceRerun = opts.Test.Rerun || opts.Cover.Rerun
	state.ShowTestOutput = opts.Test.ShowOutput || opts.Cover.ShowOutput
	state.DebugTests = debugTests
	state.ShowAllOutput = opts.OutputFlags.ShowAllOutput
	state.ParsePackageOnly = opts.ParsePackageOnly
	state.DownloadOutputs = (!opts.Build.NoDownload && !opts.Run.Remote && len(targets) > 0 && (!targets[0].IsAllSubpackages() || len(opts.BuildFlags.Include) > 0)) || opts.Build.Download
	state.SetIncludeAndExclude(opts.BuildFlags.Include, opts.BuildFlags.Exclude)
	if opts.BuildFlags.Arch.OS != "" {
		state.TargetArch = opts.BuildFlags.Arch
	}

	if state.DebugTests && len(targets) != 1 {
		log.Fatalf("-d/--debug flag can only be used with a single test target")
	}

	if opts.Run.InTempDir && opts.Run.InWD {
		log.Fatal("Can't use both --in_temp_dir and --in_wd at the same time")
	}

	runPlease(state, targets)
	return state.Successful(), state
}

func runPlease(state *core.BuildState, targets []core.BuildLabel) {
	// Acquire the lock before we start building
	if (state.NeedBuild || state.NeedTests) && !opts.FeatureFlags.NoLock {
		core.AcquireRepoLock(state)
		defer core.ReleaseRepoLock()
	}

	detailedTests := state.NeedTests && (opts.Test.Detailed || opts.Cover.Detailed ||
		(len(targets) == 1 && !targets[0].IsAllTargets() &&
			!targets[0].IsAllSubpackages() && targets[0] != core.BuildLabelStdin))
	streamTests := opts.Test.StreamResults || opts.Cover.StreamResults
	pretty := prettyOutput(opts.OutputFlags.InteractiveOutput, opts.OutputFlags.PlainOutput, opts.OutputFlags.Verbosity) && state.NeedBuild && !streamTests
	state.Cache = cache.NewCache(state)

	// Run the display
	state.Results() // important this is called now, don't ask...
	var wg sync.WaitGroup
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		output.MonitorState(ctx, state, !pretty, detailedTests, streamTests, string(opts.OutputFlags.TraceFile))
		wg.Done()
	}()
	plz.Run(targets, opts.BuildFlags.PreTargets, state, config, state.TargetArch)
	cancel()
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

// readConfig reads the initial configuration files
func readConfig(forceUpdate bool) *core.Configuration {
	cfg, err := core.ReadDefaultConfigFiles(opts.BuildFlags.Profile.Strings())
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	} else if err := cfg.ApplyOverrides(opts.BuildFlags.Option); err != nil {
		log.Fatalf("Can't override requested config setting: %s", err)
	}
	if opts.FeatureFlags.HTTPProxy != "" {
		cfg.Build.HTTPProxy = opts.FeatureFlags.HTTPProxy
	}
	config = cfg
	return cfg
}

// Runs the actual build
// Which phases get run are controlled by shouldBuild and shouldTest.
func runBuild(targets []core.BuildLabel, shouldBuild, shouldTest, isQuery bool) (bool, *core.BuildState) {
	if !isQuery {
		opts.BuildFlags.Exclude = append(opts.BuildFlags.Exclude, "manual", "manual:"+core.OsArch)
	}
	if stat, _ := os.Stdin.Stat(); (stat.Mode()&os.ModeCharDevice) == 0 && !utils.ReadingStdin(targets) {
		if len(targets) == 0 {
			// Assume they want us to read from stdin since nothing else was given.
			targets = []core.BuildLabel{core.BuildLabelStdin}
		}
	}
	if len(targets) == 0 {
		targets = core.InitialPackage()
	}
	return Please(targets, config, shouldBuild, shouldTest)
}

var originalWorkingDirectory string

// readConfigAndSetRoot reads the .plzconfig files and moves to the repo root.
func readConfigAndSetRoot(forceUpdate bool) *core.Configuration {
	if opts.BuildFlags.RepoRoot == "" {
		log.Debug("Found repo root at %s", core.MustFindRepoRoot())
	} else {
		core.RepoRoot = string(opts.BuildFlags.RepoRoot)
	}

	// Save the current working directory before moving to root
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("%s", err)
	}
	originalWorkingDirectory = wd

	// Please always runs from the repo root, so move there now.
	if err := os.Chdir(core.RepoRoot); err != nil {
		log.Fatalf("%s", err)
	}
	// Reset this now we're at the repo root.
	if opts.OutputFlags.LogFile != "" {
		if !path.IsAbs(string(opts.OutputFlags.LogFile)) {
			opts.OutputFlags.LogFile = cli.Filepath(path.Join(core.RepoRoot, string(opts.OutputFlags.LogFile)))
		}
		cli.InitFileLogging(string(opts.OutputFlags.LogFile), opts.OutputFlags.LogFileLevel, opts.OutputFlags.LogAppend)
	}
	if opts.FeatureFlags.NoHashVerification {
		log.Warning("You've disabled hash verification; this is intended to help temporarily while modifying build targets. You shouldn't use this regularly.")
	}
	config := readConfig(forceUpdate)
	// Now apply any flags that override this
	if opts.Update.Latest || opts.Update.LatestPrerelease {
		config.Please.Version.Unset()
	} else if opts.Update.Version.IsSet {
		config.Please.Version = opts.Update.Version
	}
	update.CheckAndUpdate(config, !opts.FeatureFlags.NoUpdate, forceUpdate, opts.Update.Force, !opts.Update.NoVerify, !opts.OutputFlags.PlainOutput, opts.Update.LatestPrerelease)
	return config
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
		if cli.ActiveCommand(parser.Command) == "please" && core.FindRepoRoot() {
			if config, err := core.ReadDefaultConfigFiles(nil); err == nil {
				config.PrintAliases(os.Stderr)
			}
		}
		os.Exit(0)
	}
	if opts.OutputFlags.Colour {
		cli.ShowColouredOutput = true
	} else if opts.OutputFlags.NoColour {
		cli.ShowColouredOutput = false
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
	} else if command == "help" || command == "follow" || command == "init" || command == "config" || command == "tool" {
		// These commands don't use a config file, allowing them to be run outside a repo.
		if flagsErr != nil { // This error otherwise doesn't get checked until later.
			cli.ParseFlagsFromArgsOrDie("Please", &opts, os.Args)
		}
		config = core.DefaultConfiguration()
		if command == "tool" {
			if cfg, err := core.ReadDefaultConfigFiles(opts.BuildFlags.Profile.Strings()); err == nil {
				config = cfg
			}
		}
		os.Exit(buildFunctions[command]())
	} else if opts.OutputFlags.CompletionScript {
		fmt.Printf("%s\n", string(assets.PlzComplete))
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

func unannotateLabels(als []core.AnnotatedOutputLabel) []core.BuildLabel {
	labels := make([]core.BuildLabel, len(als))
	for i, al := range als {
		labels[i] = al.BuildLabel
	}
	return labels
}

// toExitCode returns an integer process exit code based on the outcome of a build.
// 0 -> success
// 1 -> general failure (and why is he reading my hard drive?)
// 2 -> a target failed to build
// 7 -> a test failed (this is 7 for compatibility)
func toExitCode(success bool, state *core.BuildState) int {
	if success {
		return 0
	} else if state == nil {
		return 1
	} else if state.BuildFailed {
		return 2
	} else if state.TestFailed {
		if opts.Test.FailingTestsOk || opts.Cover.FailingTestsOk {
			return 0
		}
		return 7
	}
	return 1
}

func execute(command string) int {
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
	defer worker.StopAll()
	return buildFunctions[command]()
}

func main() {
	os.Exit(execute(initBuild(os.Args)))
}
