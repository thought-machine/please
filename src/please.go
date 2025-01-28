package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/thought-machine/go-flags"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/thought-machine/please/src/assets"
	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cache"
	"github.com/thought-machine/please/src/clean"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/debug"
	"github.com/thought-machine/please/src/exec"
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
	"github.com/thought-machine/please/src/process"
	"github.com/thought-machine/please/src/query"
	"github.com/thought-machine/please/src/run"
	"github.com/thought-machine/please/src/sandbox"
	"github.com/thought-machine/please/src/scm"
	"github.com/thought-machine/please/src/test"
	"github.com/thought-machine/please/src/tool"
	"github.com/thought-machine/please/src/update"
	"github.com/thought-machine/please/src/watch"
)

var log = logging.Log

var config *core.Configuration

var opts struct {
	Usage      string `usage:"Please is a high-performance multi-language build system.\n\nIt uses BUILD files to describe what to build and how to build it.\nSee https://please.build for more information about how it works and what Please can do for you."`
	BuildFlags struct {
		Config     string               `short:"c" long:"config" env:"PLZ_BUILD_CONFIG" description:"Build config to use. Defaults to opt."`
		Arch       cli.Arch             `short:"a" long:"arch" description:"Architecture to compile for."`
		RepoRoot   cli.Filepath         `short:"r" long:"repo_root" description:"Root of repository to build." env:"PLZ_REPO_ROOT"`
		NumThreads int                  `short:"n" long:"num_threads" description:"Number of concurrent build operations. Default is number of CPUs + 2."`
		Include    []string             `short:"i" long:"include" description:"Label of targets to include in automatic detection."`
		Exclude    []string             `short:"e" long:"exclude" description:"Label of targets to exclude from automatic detection."`
		Option     ConfigOverrides      `short:"o" long:"override" env:"PLZ_OVERRIDES" env-delim:";" description:"Options to override from .plzconfig (e.g. -o please.selfupdate:false)"`
		Profile    []core.ConfigProfile `long:"profile" env:"PLZ_CONFIG_PROFILE" env-delim:";" description:"Configuration profile to load; e.g. --profile=dev will load .plzconfig.dev if it exists."`
		PreTargets []core.BuildLabel    `long:"pre" hidden:"true" description:"Targets to build before the other command-line ones. Sometimes useful to debug targets generated as part of a post-build function."`
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

	BehaviorFlags struct {
		NoUpdate           bool    `long:"noupdate" description:"Disable Please attempting to auto-update itself."`
		NoHashVerification bool    `long:"nohash_verification" description:"Hash verification errors are nonfatal." env:"PLZ_NO_HASH_VERIFICATION"`
		NoLock             bool    `long:"nolock" description:"Don't attempt to lock the repo exclusively. Use with care."`
		KeepWorkdirs       bool    `long:"keep_workdirs" description:"Don't clean directories in plz-out/tmp after successfully building targets."`
		HTTPProxy          cli.URL `long:"http_proxy" env:"HTTP_PROXY" description:"HTTP proxy to use for downloads"`
		Debug              bool    `long:"debug" description:"When enabled, Please will enter into an interactive debugger when breakpoint() is called during parsing."`
		KeepGoing          bool    `long:"keep_going" description:"Continue as much as possible after an error. While the target that failed and those that depend on it cannot be build, other prerequisites of these targets can be."`
		AllowSudo          bool    `long:"allow_sudo" hidden:"true" description:"Allow running under sudo (normally this is a very bad idea)"`
	} `group:"Options that enable / disable certain behaviors"`

	HelpFlags struct {
		Help    bool `short:"h" long:"help" description:"Show this help message"`
		Version bool `long:"version" description:"Print the version of Please"`
	} `group:"Help Options"`

	Profile          string `long:"profile_file" hidden:"true" description:"Write profiling output to this file"`
	MemProfile       string `long:"mem_profile_file" hidden:"true" description:"Write a memory profile to this file"`
	MutexProfile     string `long:"mutex_profile_file" hidden:"true" description:"Write a contended mutex profile to this file"`
	GoTraceFile      string `long:"go_trace_file" hidden:"true" description:"Write a go trace profile to this file"`
	ProfilePort      int    `long:"profile_port" hidden:"true" description:"Serve profiling info on this port."`
	ParsePackageOnly bool   `description:"Parses a single package only. All that's necessary for some commands." no-flag:"true"`
	Complete         string `long:"complete" hidden:"true" env:"PLZ_COMPLETE" description:"Provide completion options for this build target."`

	Build struct {
		Shell      string `long:"shell" choice:"shell" choice:"run" optional:"true" optional-value:"shell" description:"Like --prepare, but opens a shell in the build directory with the appropriate environment variables."`
		Rebuild    bool   `long:"rebuild" description:"To force the optimisation and rebuild one or more targets."`
		NoDownload bool   `long:"nodownload" hidden:"true" description:"Don't download outputs after building. Only applies when using remote build execution."`
		Download   bool   `long:"download" hidden:"true" description:"Force download of all outputs regardless of original target spec. Only applies when using remote build execution."`
		OutDir     string `long:"out_dir" optional:"true" description:"Copies build output to given directory"`
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
		FailingTestsOk   bool         `long:"failing_tests_ok" hidden:"true" description:"Exit with status 0 even if tests fail (nonzero only if catastrophe happens)"`
		NumRuns          int          `long:"num_runs" short:"n" default:"1" description:"Number of times to run each test target."`
		Rerun            bool         `long:"rerun" description:"Rerun the test even if the hash hasn't changed."`
		Sequentially     bool         `long:"sequentially" description:"Whether to run multiple runs of the same test sequentially"`
		TestResultsFile  cli.Filepath `long:"test_results_file" default:"plz-out/log/test_results.xml" description:"File to write combined test results to."`
		SurefireDir      cli.Filepath `long:"surefire_dir" default:"plz-out/surefire-reports" description:"Directory to copy XML test results to."`
		ShowOutput       bool         `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		DebugFailingTest bool         `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest). Implies -c dbg unless otherwise set."`
		Failed           bool         `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Detailed         bool         `long:"detailed" description:"Prints more detailed output after tests."`
		Shell            string       `long:"shell" choice:"shell" choice:"run" optional:"true" optional-value:"shell" description:"Opens a shell in the test directory with the appropriate environment variables."`
		StreamResults    bool         `long:"stream_results" description:"Prints test results on stdout as they are run."`
		// Slightly awkward since we can specify a single test with arguments or multiple test targets.
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test"`
			Args   TargetsOrArgs   `positional-arg-name:"arguments" description:"Arguments or test selectors"`
		} `positional-args:"true"`
		StateArgs []string `no-flag:"true"`
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
		CoverageResultsFile cli.Filepath  `long:"coverage_results_file" env:"COVERAGE_RESULTS_FILE" default:"plz-out/log/coverage.json" description:"File to write combined coverage results to."`
		CoverageXMLReport   cli.Filepath  `long:"coverage_xml_report" env:"COVERAGE_XML_REPORT" default:"plz-out/log/coverage.xml" description:"XML File to write combined coverage results to."`
		Incremental         bool          `short:"i" long:"incremental" description:"Calculates summary statistics for incremental coverage, i.e. stats for just the lines currently modified."`
		ShowOutput          bool          `short:"s" long:"show_output" description:"Always show output of tests, even on success."`
		DebugFailingTest    bool          `short:"d" long:"debug" description:"Allows starting an interactive debugger on test failure. Does not work with all test types (currently only python/pytest). Implies -c dbg unless otherwise set."`
		Failed              bool          `short:"f" long:"failed" description:"Runs just the test cases that failed from the immediately previous run."`
		Detailed            bool          `long:"detailed" description:"Prints more detailed output after tests."`
		Shell               string        `long:"shell" choice:"shell" choice:"run" optional:"true" optional-value:"shell" description:"Opens a shell in the test directory with the appropriate environment variables."`
		StreamResults       bool          `long:"stream_results" description:"Prints test results on stdout as they are run."`
		Args                struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to test"`
			Args   TargetsOrArgs   `positional-arg-name:"arguments" description:"Arguments or test selectors"`
		} `positional-args:"true"`
	} `command:"cover" description:"Builds and tests one or more targets, and calculates coverage."`

	Debug struct {
		Port  int `short:"p" long:"port" description:"Debugging server port"`
		Share struct {
			Network bool `long:"share_network" description:"Share network namespace"`
			Mount   bool `long:"share_mount" description:"Share mount namespace"`
		} `group:"Options to override mount and network namespacing on linux, if configured"`
		Env  map[string]string `short:"e" long:"env" description:"Environment variables to set for the debugged process"`
		Args struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to debug"`
			Args   []string        `positional-arg-name:"arguments" description:"Arguments to pass to target"`
		} `positional-args:"true"`
	} `command:"debug" description:"Starts a debug session on the given target if supported by its build definition."`

	Run struct {
		Env        bool   `long:"env" description:"Overrides environment variables (e.g. PATH) in the new process."`
		Rebuild    bool   `long:"rebuild" description:"To force the optimisation and rebuild one or more targets."`
		InWD       bool   `long:"in_wd" description:"Deprecated in favour of --wd=/path/to/this/directory. When running locally, stay in the original working directory."`
		WD         string `long:"wd" description:"The working directory in which to run the target."`
		InTempDir  bool   `long:"in_tmp_dir" description:"Runs in a temp directory, setting env variables and copying in runtime data similar to tests."`
		EntryPoint string `long:"entry_point" short:"e" description:"The entry point of the target to use." default:""`
		Cmd        string `long:"cmd" description:"Overrides the command to be run. This is useful when the initial command needs to be wrapped in another one." default:""`
		Parallel   struct {
			NumTasks       int                `short:"n" long:"num_tasks" default:"10" description:"Maximum number of subtasks to run in parallel"`
			Output         process.OutputMode `long:"output" default:"default" choice:"default" choice:"quiet" choice:"group_immediate" description:"Allows to control how the output should be handled."`
			PositionalArgs struct {
				Targets TargetsOrArgs `positional-arg-name:"target" required:"true" description:"Target to run"`
			} `positional-args:"true" required:"true"`
			Args   cli.Filepaths `short:"a" long:"arg" description:"Arguments to pass to the target. Deprecated, pass them directly as arguments (after -- if needed)"`
			Detach bool          `long:"detach" description:"Detach from the parent process when all children have spawned"`
		} `command:"parallel" description:"Runs a sequence of targets in parallel"`
		Sequential struct {
			Quiet          bool               `short:"q" long:"quiet" description:"Suppress output from successful subprocesses."`
			Output         process.OutputMode `long:"output" default:"default" choice:"default" choice:"quiet" choice:"group_immediate" description:"Allows to control how the output should be handled."`
			PositionalArgs struct {
				Targets TargetsOrArgs `positional-arg-name:"target" required:"true" description:"Target to run"`
			} `positional-args:"true" required:"true"`
			Args cli.Filepaths `short:"a" long:"arg" description:"Arguments to pass to the target. Deprecated, pass them directly as arguments (after -- if needed)"`
		} `command:"sequential" description:"Runs a sequence of targets sequentially."`
		Args struct {
			Target core.AnnotatedOutputLabel `positional-arg-name:"target" required:"true" description:"Target to run"`
			Args   cli.Filepaths             `positional-arg-name:"arguments" description:"Arguments to pass to target when running (to pass flags to the target, put -- before them)"`
		} `positional-args:"true"`
		Remote bool `long:"remote" description:"Send targets to be executed remotely."`
	} `command:"run" subcommands-optional:"true" description:"Builds and runs a single target"`

	Exec struct {
		Output struct {
			OutputPath string   `long:"output_path" description:"The path to the directory to save outputs into" default:"."`
			Output     []string `long:"out" description:"A file or folder relative to the working directory to save to the output path"`
		} `group:"Options controlling what files to save from the working directory and where to save them"`
		Env   map[string]string `short:"e" long:"env" description:"Environment variables to set in the execution environment"`
		Share struct {
			Network bool `long:"share_network" description:"Share network namespace"`
			Mount   bool `long:"share_mount" description:"Share mount namespace"`
		} `group:"Options to override mount and network namespacing on linux, if configured"`
		Args struct {
			Target core.AnnotatedOutputLabel `positional-arg-name:"target" required:"true" description:"Target to execute"`
			Args   []string                  `positional-arg-name:"arg" description:"Arguments to the executed command"`
		} `positional-args:"true"`
		Sequential struct {
			Args struct {
				Targets TargetsOrArgs `positional-arg-name:"target" required:"true" description:"Targets to execute, or arguments to them"`
			} `positional-args:"true"`
			Output process.OutputMode `long:"output" default:"default" choice:"default" choice:"quiet" choice:"group_immediate" description:"Controls how output from subprocesses is handled."`
		} `command:"sequential" description:"Execute a series of targets sequentially"`
		Parallel struct {
			NumTasks int `short:"n" long:"num_tasks" default:"10" description:"Maximum number of subtasks to run in parallel"`
			Args     struct {
				Targets TargetsOrArgs `positional-arg-name:"target" required:"true" description:"Targets to execute, or arguments to them"`
			} `positional-args:"true"`
			Output process.OutputMode `long:"output" default:"default" choice:"default" choice:"quiet" choice:"group_immediate" description:"Controls how output from subprocesses is handled."`
			Update cli.Duration       `long:"update" default:"10s" description:"Frequency to log updates on running subprocesses. Has no effect for 'default' output mode."`
		} `command:"parallel" description:"Execute a number of targets in parallel"`
	} `command:"exec" subcommands-optional:"true" description:"Executes a single target in a hermetic build environment"`

	Clean struct {
		NoBackground bool     `long:"nobackground" short:"f" description:"Don't fork & detach until clean is finished."`
		Rm           string   `long:"rm" hidden:"true" description:"Removes a specific directory. Only used internally to do async removals."`
		Args         struct { // Inner nesting is necessary to make positional-args work :(
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to clean (default is to clean everything)"`
		} `positional-args:"true"`
	} `command:"clean" description:"Cleans build artifacts" subcommands-optional:"true"`

	Watch struct {
		Run    bool `short:"r" long:"run" description:"Runs the specified targets when they change (default is to build or test as appropriate)."`
		NoTest bool `long:"notest" description:"If set, no tests will be ran. The targets will only be re-built."`
		Args   struct {
			Target core.BuildLabel `positional-arg-name:"target" description:"Target to watch for changes"`
			Args   TargetsOrArgs   `positional-arg-name:"arguments" description:"Additional targets to watch, or test selectors"`
		} `positional-args:"true" required:"true"`
	} `command:"watch" description:"Watches sources of targets for changes and rebuilds them"`

	Update struct {
		Force            bool        `long:"force" description:"Forces a re-download of the new version."`
		NoVerify         bool        `long:"noverify" description:"Skips signature and hash verification of downloaded version"`
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
		} `command:"config" description:"Initialises specific attributes of config files. Warning, will add duplicate entries if attribute already set"`
		Pleasings struct {
			Revision  string `short:"r" long:"revision" description:"The revision to pin the pleasings repo to. This can be a branch, commit, tag, or other git reference."`
			Location  string `short:"l" long:"location" description:"The location of the build file to write the subrepo rule to" default:"BUILD"`
			PrintOnly bool   `long:"print" description:"Print the rule to standard out instead of writing it to a file"`
		} `command:"pleasings" description:"Initialises the pleasings repo"`
		Pleasew struct {
		} `command:"pleasew" description:"Initialises the pleasew wrapper script"`
		Plugin struct {
			Version string `short:"v" long:"version" description:"Version of plugin to install. If not set, the latest is found."`
			Args    struct {
				Plugins []string `positional-arg-name:"plugin" required:"true" description:"Plugins to install"`
			} `positional-args:"true"`
		} `command:"plugin" hidden:"true" description:"Install a plugin and migrate any language-specific config values"`
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
		NoTrim bool   `long:"notrim" description:"export the build file as is, without trying to trim unused targets"`
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
			Tool tool.Tool     `positional-arg-name:"tool" description:"Tool to invoke (arcat, lint, etc)"`
			Args cli.Filepaths `positional-arg-name:"arguments" description:"Arguments to pass to the tool"`
		} `positional-args:"true"`
	} `command:"tool" hidden:"true" description:"Invoke one of Please's sub-tools"`

	Query struct {
		Deps struct {
			DOT    bool `long:"dot" description:"Output in dot format"`
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
			Except []core.BuildLabel `long:"except" description:"Targets to exclude from path calculation"`
			Hidden bool              `long:"hidden" description:"Show hidden targets as well"`
			Args   struct {
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
			JSON       bool     `long:"json" description:"Print the targets as json rather than python"`
			OmitHidden bool     `long:"omit_hidden" description:"Omit hidden fields. Can be useful when using wildcard"`
			Fields     []string `short:"f" long:"field" description:"Individual fields to print of the target"`
			Labels     []string `short:"l" long:"label" description:"Prints all labels with the given prefix (with the prefix stripped off). Overrides --field."`
			Args       struct {
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
			JSON bool `long:"json" description:"Print the outputs as a json map from target to lists of output files, rather than a flat list of files"`
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to display outputs for" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"output" alias:"outputs" description:"Prints all outputs of a target."`
		Graph struct {
			Args struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to render graph for"`
			} `positional-args:"true"`
		} `command:"graph" description:"Prints a representation of the build graph."`
		WhatInputs struct {
			Hidden        bool `long:"hidden" short:"h" description:"Output internal / hidden targets too."`
			EchoFiles     bool `long:"echo_files" description:"Echo the file for which the printed output is responsible."`
			IgnoreUnknown bool `long:"ignore_unknown" description:"Ignore any files that are not inputs to existing build targets"`
			Args          struct {
				Files cli.StdinStrings `positional-arg-name:"files" description:"Files to query as sources to targets" required:"true"`
			} `positional-args:"true" required:"true"`
		} `command:"whatinputs" description:"Prints out target(s) with provided file(s) as inputs"`
		WhatOutputs struct {
			EchoFiles bool `long:"echo_files" description:"Echo the file for which the printed output is responsible."`
			Args      struct {
				Files cli.StdinStrings `positional-arg-name:"files" required:"true" description:"Files to query targets responsible for"`
			} `positional-args:"true"`
		} `command:"whatoutputs" description:"Prints out target(s) responsible for outputting provided file(s)"`
		Rules struct {
			Args struct {
				Files cli.StdinStrings `positional-arg-name:"files" description:"Files to parse for build rules." hidden:"true"`
			} `positional-args:"true"`
		} `command:"rules" description:"Prints built-in rules to stdout as JSON"`
		Changes struct {
			Since            string `short:"s" long:"since" default:"origin/master" description:"Revision to compare against"`
			IncludeDependees string `long:"include_dependees" default:"none" choice:"none" choice:"direct" choice:"transitive" description:"Deprecated: use level 1 for direct and -1 for transitive. Include direct or transitive dependees of changed targets."`
			IncludeSubrepos  bool   `long:"include_subrepos" description:"Include changed targets that belong to subrepos."`
			Level            int    `long:"level" default:"-2" description:"Levels of the dependencies of changed targets (-1 for unlimited)." default-mask:"0"`
			Inexact          bool   `long:"inexact" description:"Calculate changes more quickly and without doing any SCM checkouts, but may miss some targets."`
			In               string `long:"in" description:"Calculate changes contained within given scm spec (commit range/sha/ref/etc). Implies --inexact."`
			Args             struct {
				Files cli.StdinStrings `positional-arg-name:"files" description:"Files to calculate changes for. Overrides flags relating to SCM operations."`
			} `positional-args:"true"`
		} `command:"changes" description:"Calculates the set of changed targets in regard to a set of modified files or SCM commits."`
		Filter struct {
			Hidden bool `long:"hidden" description:"Show hidden targets as well"`
			Args   struct {
				Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to filter"`
			} `positional-args:"true"`
		} `command:"filter" description:"Filter the given set of targets according to some rules"`
		RepoRoot struct {
		} `command:"reporoot" alias:"repo_root" description:"Output the root of the current Please repo"`
		Config struct {
			JSON bool `long:"json" description:"Output as JSON."`
			Args struct {
				Options []string `positional-arg-name:"options" description:"Print specific options."`
			} `positional-args:"true"`
		} `command:"config" description:"Prints the configuration settings"`
	} `command:"query" description:"Queries information about the build state"`
	Generate struct {
		Gitignore string `long:"update_gitignore" description:"The gitignore file to write the generated sources to"`
		Args      struct {
			Targets []core.BuildLabel `positional-arg-name:"targets" description:"Targets to filter"`
		} `positional-args:"true"`
	} `command:"generate" description:"Builds all code generation targets in the repository and prints the generated files."`
}

// Definitions of what we do for each command.
// Functions are called after args are parsed and return a POSIX exit code (0 means success).
var buildFunctions = map[string]func() int{
	"build": func() int {
		success, state := runBuild(opts.Build.Args.Targets, true, false, false)
		if !success || opts.Build.OutDir == "" {
			return toExitCode(success, state)
		}
		for _, label := range state.ExpandOriginalLabels() {
			target := state.Graph.TargetOrDie((label))
			for _, out := range target.Outputs() {
				from := filepath.Join(target.OutDir(), out)
				fm, err := os.Lstat(from)
				if err != nil {
					log.Fatalf("Failed to get file mode on build output files: %s", err)
				}
				err = fs.CopyFile(from, filepath.Join(opts.Build.OutDir, target.PackageDir(), out), fm.Mode())
				if err != nil {
					log.Fatalf("Failed to output build to provided directory: %s", err)
				}
			}
		}
		return 0
	},
	"hash": func() int {
		if opts.Hash.Update {
			opts.BehaviorFlags.NoHashVerification = true
		}
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
		targets, args := testTargets(opts.Test.Args.Target, opts.Test.Args.Args, opts.Test.Failed, opts.Test.TestResultsFile)
		success, state := doTest(targets, args, opts.Test.SurefireDir, opts.Test.TestResultsFile)
		return toExitCode(success, state)
	},
	"cover": func() int {
		opts.Cover.active = true
		if opts.BuildFlags.Config != "" {
			log.Warning("Build config overridden; coverage may not be available for some languages")
		} else {
			opts.BuildFlags.Config = "cover"
		}
		targets, args := testTargets(opts.Cover.Args.Target, opts.Cover.Args.Args, opts.Cover.Failed, opts.Cover.TestResultsFile)
		fs.RemoveAll(string(opts.Cover.CoverageResultsFile))
		success, state := doTest(targets, args, opts.Cover.SurefireDir, opts.Cover.TestResultsFile)
		test.AddOriginalTargetsToCoverage(state, opts.Cover.IncludeAllFiles)
		test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension, state.Config.Cover.ExcludeGlob)

		var stats *test.IncrementalStats
		if opts.Cover.Incremental {
			lines, err := scm.NewFallback(core.RepoRoot).ChangedLines()
			if err != nil {
				log.Fatalf("Failed to determine changes: %s", err)
			}
			stats = test.CalculateIncrementalStats(state, lines)
		}
		if opts.Cover.CoverageResultsFile != "" {
			test.WriteCoverageToFileOrDie(state.Coverage, string(opts.Cover.CoverageResultsFile), stats)
		}
		if opts.Cover.CoverageXMLReport != "" {
			test.WriteXMLCoverageToFileOrDie(targets, state.Coverage, string(opts.Cover.CoverageXMLReport))
		}

		if opts.Cover.LineCoverageReport && success {
			output.PrintLineCoverageReport(state, opts.Cover.IncludeFile.AsStrings())
		} else if !opts.Cover.NoCoverageReport && opts.Cover.Shell == "" {
			output.PrintCoverage(state, opts.Cover.IncludeFile.AsStrings())
		}
		if opts.Cover.Incremental {
			output.PrintIncrementalCoverage(stats)
		}
		return toExitCode(success, state)
	},
	"debug": func() int {
		success, state := runBuild([]core.BuildLabel{opts.Debug.Args.Target}, true, false, false)
		if !success {
			return toExitCode(success, state)
		}
		return debug.Debug(state, opts.Debug.Args.Target, opts.Debug.Args.Args, exec.ConvertEnv(opts.Debug.Env), opts.Debug.Share.Network, opts.Debug.Share.Mount)
	},
	"exec": func() int {
		success, state := runBuild([]core.BuildLabel{opts.Exec.Args.Target.BuildLabel}, true, false, false)
		if !success {
			return toExitCode(success, state)
		}

		target := state.Graph.TargetOrDie(opts.Exec.Args.Target.BuildLabel)
		dir := target.ExecDir()
		shouldSandbox := target.Sandbox
		if code := exec.Exec(state, opts.Exec.Args.Target, dir, exec.ConvertEnv(opts.Exec.Env), nil, opts.Exec.Args.Args, false, process.NewSandboxConfig(shouldSandbox && !opts.Exec.Share.Network, shouldSandbox && !opts.Exec.Share.Mount)); code != 0 {
			return code
		}

		for _, out := range opts.Exec.Output.Output {
			from := filepath.Join(dir, out)
			to := filepath.Join(opts.Exec.Output.OutputPath, out)

			if err := fs.EnsureDir(to); err != nil {
				log.Fatalf("%v", err)
			}

			if err := fs.RecursiveLink(from, to); err != nil {
				log.Fatalf("failed to move output: %v", err)
			}
		}
		return 0
	},
	"exec.sequential": func() int {
		annotated, unannotated, args := opts.Exec.Sequential.Args.Targets.Separate(true)
		if len(unannotated) == 0 {
			return 0
		}
		success, state := runBuild(unannotated, true, false, false)
		if !success {
			return toExitCode(success, state)
		}
		if code := exec.Sequential(state, opts.Exec.Sequential.Output, annotated, exec.ConvertEnv(opts.Exec.Env), args, opts.Exec.Share.Network, opts.Exec.Share.Mount); code != 0 {
			return code
		}
		return 0
	},
	"exec.parallel": func() int {
		annotated, unannotated, args := opts.Exec.Parallel.Args.Targets.Separate(true)
		if len(unannotated) == 0 {
			return 0
		}
		success, state := runBuild(unannotated, true, false, false)
		if !success {
			return toExitCode(success, state)
		}
		if code := exec.Parallel(state, opts.Exec.Parallel.Output, time.Duration(opts.Exec.Parallel.Update), annotated, exec.ConvertEnv(opts.Exec.Env), args, opts.Exec.Parallel.NumTasks, opts.Exec.Share.Network, opts.Exec.Share.Mount); code != 0 {
			return code
		}
		return 0
	},
	"run": func() int {
		if success, state := runBuild([]core.BuildLabel{opts.Run.Args.Target.BuildLabel}, true, false, false); success {
			var dir string
			if opts.Run.WD != "" {
				dir = getAbsolutePath(opts.Run.WD, originalWorkingDirectory)
			}

			if opts.Run.EntryPoint != "" {
				opts.Run.Args.Target.Annotation = opts.Run.EntryPoint
			}

			annotatedOutputLabels := state.ExpandOriginalMaybeAnnotatedLabels([]core.AnnotatedOutputLabel{opts.Run.Args.Target})
			if len(annotatedOutputLabels) != 1 {
				log.Fatalf("%v expanded to more than one target. If you want to run multiple targets, use `plz run parallel %v` or `plz run sequential %v`. ", opts.Run.Args.Target, opts.Run.Args.Target, opts.Run.Args.Target)
			}

			run.Run(state, annotatedOutputLabels[0], opts.Run.Args.Args.AsStrings(), opts.Run.Remote, opts.Run.Env, opts.Run.InTempDir, dir, opts.Run.Cmd)
		}
		return 1 // We should never return from run.Run so if we make it here something's wrong.
	},
	"run.parallel": func() int {
		annotated, unannotated, args := opts.Run.Parallel.PositionalArgs.Targets.Separate(true)
		if len(unannotated) == 0 {
			return 0
		}
		if success, state := runBuild(unannotated, true, false, false); success {
			var dir string
			if opts.Run.WD != "" {
				dir = getAbsolutePath(opts.Run.WD, originalWorkingDirectory)
			}
			output := opts.Run.Parallel.Output
			args = append(args, opts.Run.Parallel.Args.AsStrings()...)
			annotated = state.ExpandMaybeAnnotatedLabels(annotated)
			os.Exit(run.Parallel(context.Background(), state, annotated, args, opts.Run.Parallel.NumTasks, output, opts.Run.Remote, opts.Run.Env, opts.Run.Parallel.Detach, opts.Run.InTempDir, dir))
		}
		return 1
	},
	"run.sequential": func() int {
		annotated, unannotated, args := opts.Run.Sequential.PositionalArgs.Targets.Separate(true)
		if len(unannotated) == 0 {
			return 0
		}
		if success, state := runBuild(unannotated, true, false, false); success {
			var dir string
			if opts.Run.WD != "" {
				dir = getAbsolutePath(opts.Run.WD, originalWorkingDirectory)
			}
			output := opts.Run.Sequential.Output
			if opts.Run.Sequential.Quiet {
				log.Warningf("--quiet has been deprecated in favour of --output=quiet and will be removed in v17.")
				output = process.Quiet
			}
			args = append(args, opts.Run.Sequential.Args.AsStrings()...)
			annotated = state.ExpandMaybeAnnotatedLabels(annotated)
			os.Exit(run.Sequential(state, annotated, args, output, opts.Run.Remote, opts.Run.Env, opts.Run.InTempDir, dir))
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
		cmd := core.ReadPreviousOperationOrDie()
		log.Notice("OP PLZ: %s", strings.Join(cmd, " "))
		// Annoyingly we don't seem to have any access to execvp() which would be rather useful here...
		executable, err := os.Executable()
		if err == nil {
			err = syscall.Exec(executable, append([]string{executable}, cmd...), os.Environ())
		}
		log.Fatalf("SORRY OP: %s", err) // On success Run never returns.
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
		if opts.Format.Quiet && opts.Format.Write {
			log.Fatal("Can't use both --quiet and --write at the same time")
		}
		if changed, err := format.Format(config, opts.Format.Args.Files.AsStrings(), opts.Format.Write, opts.Format.Quiet); err != nil {
			log.Fatalf("Failed to reformat files: %s", err)
		} else if changed && opts.Format.Quiet {
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
		fmt.Println("There are auxiliary build rules available via plugins for various languages and technologies.")
		fmt.Println("For a list of available plugins visit https://github.com/please-build/please-rules. Or to get up and running quickly, try `plz init plugin [go|java|python|cc]`")
		fmt.Println()

		return 0
	},
	"init.config": func() int {
		if opts.Init.Config.User {
			plzinit.InitConfigFile(fs.ExpandHomePath(core.UserConfigFileName), opts.Init.Config.Args.Options)
		} else if opts.Init.Config.Local {
			plzinit.InitConfigFile(core.LocalConfigFileName, opts.Init.Config.Args.Options)
		} else {
			plzinit.InitConfigFile(core.ConfigFileName, opts.Init.Config.Args.Options)
		}
		return 0
	},
	"init.pleasings": func() int {
		if err := plzinit.InitPleasings(opts.Init.Pleasings.Location, opts.Init.Pleasings.PrintOnly, opts.Init.Pleasings.Revision); err != nil {
			log.Fatalf("failed to write pleasings subrepo file: %v", err)
		}
		return 0
	},
	"init.pleasew": func() int {
		plzinit.InitWrapperScript()
		return 0
	},
	"init.plugin": func() int {
		if err := plzinit.InitPlugins(opts.Init.Plugin.Args.Plugins, opts.Init.Plugin.Version); err != nil {
			log.Fatalf("%s", err)
		}
		return 0
	},
	"export": func() int {
		success, state := runBuild(opts.Export.Args.Targets, false, false, false)
		if success {
			export.ToDir(state, opts.Export.Output, opts.Export.NoTrim, state.ExpandOriginalLabels())
		}
		return toExitCode(success, state)
	},
	"export.outputs": func() int {
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
		return runTool(opts.Tool.Args.Tool)
	},
	"query.deps": func() int {
		return runQuery(true, opts.Query.Deps.Args.Targets, func(state *core.BuildState) {
			query.Deps(os.Stdout, state, state.ExpandOriginalLabels(), opts.Query.Deps.Hidden, opts.Query.Deps.Level, opts.Query.Deps.DOT)
		})
	},
	"query.revdeps": func() int {
		labels := plz.ReadStdinLabels(opts.Query.ReverseDeps.Args.Targets)
		return runQuery(true, append(labels, core.WholeGraph...), func(state *core.BuildState) {
			query.ReverseDeps(state, state.ExpandLabels(labels), opts.Query.ReverseDeps.Level, opts.Query.ReverseDeps.Hidden)
		})
	},
	"query.somepath": func() int {
		a := plz.ReadStdinLabels([]core.BuildLabel{opts.Query.SomePath.Args.Target1})
		b := plz.ReadStdinLabels([]core.BuildLabel{opts.Query.SomePath.Args.Target2})
		return runQuery(true, append(a, b...), func(state *core.BuildState) {
			if err := query.SomePath(state.Graph, a, b, opts.Query.SomePath.Except, opts.Query.SomePath.Hidden); err != nil {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}
		})
	},
	"query.alltargets": func() int {
		return runQuery(true, opts.Query.AllTargets.Args.Targets, func(state *core.BuildState) {
			query.AllTargets(state.Graph, state.ExpandOriginalLabels(), opts.Query.AllTargets.Hidden)
		})
	},
	"query.print": func() int {
		return runQuery(false, opts.Query.Print.Args.Targets, func(state *core.BuildState) {
			query.Print(state, state.ExpandOriginalLabels(), opts.Query.Print.Fields, opts.Query.Print.Labels, opts.Query.Print.OmitHidden, opts.Query.Print.JSON)
		})
	},
	"query.input": func() int {
		return runQuery(true, opts.Query.Input.Args.Targets, func(state *core.BuildState) {
			query.TargetInputs(state.Graph, state.ExpandOriginalLabels())
		})
	},
	"query.output": func() int {
		return runQuery(true, opts.Query.Output.Args.Targets, func(state *core.BuildState) {
			query.TargetOutputs(state.Graph, state.ExpandOriginalLabels(), opts.Query.Output.JSON)
		})
	},
	"query.completions": func() int {
		// Somewhat fiddly because the inputs are not necessarily well-formed at this point.
		opts.ParsePackageOnly = true
		fragments := opts.Query.Completions.Args.Fragments.Get()
		if opts.Query.Completions.Cmd == "help" {
			// Special-case completing help topics rather than build targets.
			if len(fragments) == 0 {
				help.Topics("", config)
			} else {
				help.Topics(fragments[0], config)
			}
			return 0
		}

		var qry string
		if len(fragments) == 0 {
			qry = "//"
		} else {
			qry = fragments[0]
		}

		completions, labels := getCompletions(qry)

		// Rerun the completions if we didn't match any labels and matched just one package
		for len(completions.Pkgs) == 1 && len(labels) == 0 {
			oldPackage := completions.Pkgs[0]
			completions, labels = getCompletions("//" + completions.Pkgs[0])
			// We really matched no labels so we should stop
			if len(completions.Pkgs) == 1 && completions.Pkgs[0] == oldPackage {
				break
			}
		}

		abs := strings.HasPrefix(qry, "//")
		for _, l := range labels {
			query.PrintCompletion(l, abs)
		}
		for _, p := range completions.Pkgs {
			query.PrintCompletion(p, abs)
		}

		return 0
	},
	"query.graph": func() int {
		targets := opts.Query.Graph.Args.Targets
		return runQuery(true, targets, func(state *core.BuildState) {
			if len(opts.Query.Graph.Args.Targets) == 0 {
				targets = opts.Query.Graph.Args.Targets // It special-cases doing the full graph.
			}
			query.Graph(state, state.ExpandLabels(targets))
		})
	},
	"query.whatinputs": func() int {
		files := opts.Query.WhatInputs.Args.Files.Get()
		// Make all these relative to the repo root; many things do not work if they're absolute.
		for i, file := range files {
			if filepath.IsAbs(file) {
				rel, err := filepath.Rel(core.RepoRoot, file)
				if err != nil {
					log.Fatalf("Failed to make input relative to repo root: %s", err)
				} else if strings.HasPrefix(rel, "..") {
					log.Fatalf("Input %s does not lie within this repo (relative path: %s)", file, rel)
				}
				files[i] = rel
			}
		}
		// We only need this to retrieve the BuildFileName
		state := core.NewBuildState(config)
		labels := make([]core.BuildLabel, 0, len(files))
		for _, file := range files {
			labels = append(labels, core.FindOwningPackage(state, file))
		}
		return runQuery(true, labels, func(state *core.BuildState) {
			query.WhatInputs(state.Graph, files, opts.Query.WhatInputs.Hidden, opts.Query.WhatInputs.EchoFiles, opts.Query.WhatInputs.IgnoreUnknown)
		})
	},
	"query.whatoutputs": func() int {
		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
			query.WhatOutputs(state.Graph, opts.Query.WhatOutputs.Args.Files.Get(), opts.Query.WhatOutputs.EchoFiles)
		})
	},
	"query.rules": func() int {
		help.PrintRuleArgs(opts.Query.Rules.Args.Files)
		return 0
	},
	"query.changes": func() int {
		// query changes always excludes 'manual' targets.
		opts.BuildFlags.Exclude = append(opts.BuildFlags.Exclude, "manual", "manual:"+core.OsArch)
		includeSubrepos := opts.Query.Changes.IncludeSubrepos
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
				for _, target := range query.Changes(state, files, level, includeSubrepos) {
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
		original := scm.CurrentRevIdentifier(false)
		files := scm.ChangedFiles(opts.Query.Changes.Since, true, "")
		log.Debugf("Number of changed files: %d", len(files))
		if err := scm.Checkout(opts.Query.Changes.Since); err != nil {
			log.Fatalf("%s", err)
		}
		readConfig()
		_, before := runBuild(core.WholeGraph, false, false, false)
		// N.B. Ignore failure here; if we can't parse the graph before then it will suffice to
		//      assume that anything we don't know about has changed.
		if err := scm.Checkout(original); err != nil {
			log.Fatalf("%s", err)
		}
		readConfig()
		success, after := runBuild(core.WholeGraph, false, false, false)
		if !success {
			return 1
		}
		for _, target := range query.DiffGraphs(before, after, files, level, includeSubrepos) {
			fmt.Println(target.String())
		}
		return 0
	},
	"query.filter": func() int {
		return runQuery(false, opts.Query.Filter.Args.Targets, func(state *core.BuildState) {
			query.Filter(state, state.ExpandOriginalLabels(), opts.Query.Filter.Hidden)
		})
	},
	"query.reporoot": func() int {
		fmt.Println(core.RepoRoot)
		return 0
	},
	"query.config": func() int {
		if opts.Query.Config.JSON {
			if len(opts.Query.Config.Args.Options) > 0 {
				log.Fatal("The --option flag isn't available with the --json flag")
			}
			query.ConfigJSON(config)
		} else {
			query.Config(config, opts.Query.Config.Args.Options)
		}
		return 0
	},
	"watch": func() int {
		targets, args := testTargets(opts.Watch.Args.Target, opts.Watch.Args.Args, false, "")
		// Don't ask it to test now since we don't know if any of them are tests yet.
		success, state := runBuild(targets, true, false, false)
		state.NeedRun = opts.Watch.Run
		watch.Watch(state, state.ExpandOriginalLabels(), args, opts.Watch.NoTest, runPlease)
		return toExitCode(success, state)
	},
	"generate": func() int {
		opts.BuildFlags.Include = append(opts.BuildFlags.Include, "codegen")

		if opts.Generate.Gitignore != "" {
			pkg := filepath.Dir(opts.Generate.Gitignore)
			if pkg == "." {
				pkg = ""
			}
			target := core.BuildLabel{
				PackageName: pkg,
				Name:        "...",
			}

			if len(opts.Generate.Args.Targets) != 0 {
				log.Warning("You've provided targets, and a gitignore to update. Ignoring the provided targets and building %v", target)
			}

			opts.Generate.Args.Targets = []core.BuildLabel{target}
		}

		if success, state := runBuild(opts.Generate.Args.Targets, true, false, true); success {
			if opts.Generate.Gitignore != "" {
				err := generate.UpdateGitignore(state.Graph, state.ExpandOriginalLabels(), opts.Generate.Gitignore)
				if err != nil {
					log.Fatalf("failed to update gitignore: %v", err)
				}
			}

			// This may seem counterintuitive but if this was set, we would've linked during the build.
			// If we've opted to not automatically link generated sources during the build, we should link them now.
			if !state.Config.ShouldLinkGeneratedSources() {
				generate.LinkGeneratedSources(state, state.ExpandOriginalLabels())
			}
			return 0
		}
		return 1
	},
	"sandbox": func() int {
		if err := sandbox.Sandbox(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return 0
	},
}

// Check if tool is given as label or path and then run
func runTool(_tool tool.Tool) int {
	c := core.DefaultConfiguration()
	if cfg, err := core.ReadDefaultConfigFiles(fs.HostFS, opts.BuildFlags.Profile); err == nil {
		c = cfg
	}
	t, _ := tool.MatchingTool(c, string(_tool))

	if !core.LooksLikeABuildLabel(t) {
		tool.Run(c, tool.Tool(t), opts.Tool.Args.Args.AsStrings())
	}

	label := core.ParseBuildLabels([]string{t})

	// We skip loading the repo config in init for `plz tool` to allow this command to work outside of a repo root. If
	// the tool looks like a build label, we need to set the repo root now.
	config = mustReadConfigAndSetRoot(false)
	if success, state := runBuild(label, true, false, false); success {
		annotatedOutputLabels := core.AnnotateLabels(label)
		run.Run(state, annotatedOutputLabels[0], opts.Tool.Args.Args.AsStrings(), false, false, false, "", "")
	}
	// If all went well, we shouldn't get here.
	return 1
}

// ConfigOverrides are used to implement completion on the -o flag.
type ConfigOverrides map[string]string

// Complete implements the flags.Completer interface.
func (overrides ConfigOverrides) Complete(match string) []flags.Completion {
	return core.DefaultConfiguration().Completions(match)
}

// Get an absolute path from a relative path.
func getAbsolutePath(path string, here string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(here, path)
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

func doTest(targets []core.BuildLabel, args []string, surefireDir cli.Filepath, resultsFile cli.Filepath) (bool, *core.BuildState) {
	fs.RemoveAll(string(surefireDir))
	fs.RemoveAll(string(resultsFile))
	os.MkdirAll(string(surefireDir), core.DirPermissions)
	opts.Test.StateArgs = args
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
	debug := !opts.Debug.Args.Target.IsEmpty()
	debugFailingTests := opts.Test.DebugFailingTest || opts.Cover.DebugFailingTest
	if opts.BuildFlags.Config != "" {
		config.Build.Config = opts.BuildFlags.Config
	} else if debug || debugFailingTests {
		config.Build.Config = "dbg"
	}
	state := core.NewBuildState(config)
	state.KeepGoing = opts.BehaviorFlags.KeepGoing
	state.VerifyHashes = !opts.BehaviorFlags.NoHashVerification
	// Only one of these two can be passed
	state.NumTestRuns = uint16(opts.Test.NumRuns)
	if opts.Cover.NumRuns > opts.Test.NumRuns {
		state.NumTestRuns = uint16(opts.Cover.NumRuns)
	}
	state.TestSequentially = opts.Test.Sequentially || opts.Cover.Sequentially // Similarly here.
	state.TestArgs = opts.Test.StateArgs
	state.NeedCoverage = opts.Cover.active
	state.NeedBuild = shouldBuild
	state.NeedTests = shouldTest
	state.NeedRun = !opts.Run.Args.Target.IsEmpty() || len(opts.Run.Parallel.PositionalArgs.Targets) > 0 || len(opts.Run.Sequential.PositionalArgs.Targets) > 0 || !opts.Exec.Args.Target.IsEmpty() || len(opts.Exec.Sequential.Args.Targets) > 0 || len(opts.Exec.Parallel.Args.Targets) > 0 || opts.Tool.Args.Tool != "" || debug
	state.NeedHashesOnly = len(opts.Hash.Args.Targets) > 0
	state.PrepareOnly = opts.Build.Shell != "" || opts.Test.Shell != "" || opts.Cover.Shell != ""
	state.Watch = !opts.Watch.Args.Target.IsEmpty()
	state.CleanWorkdirs = !opts.BehaviorFlags.KeepWorkdirs
	state.ForceRebuild = opts.Build.Rebuild || opts.Run.Rebuild
	state.ForceRerun = opts.Test.Rerun || opts.Cover.Rerun
	state.ShowTestOutput = opts.Test.ShowOutput || opts.Cover.ShowOutput
	state.DebugPort = opts.Debug.Port
	state.DebugFailingTests = debugFailingTests
	state.ShowAllOutput = opts.OutputFlags.ShowAllOutput
	state.ParsePackageOnly = opts.ParsePackageOnly
	state.EnableBreakpoints = opts.BehaviorFlags.Debug

	// What outputs get downloaded in remote execution.
	if debug {
		state.OutputDownload = core.TransitiveOutputDownload
	} else if (!opts.Build.NoDownload && !opts.Run.Remote && len(targets) > 0 && (!targets[0].IsAllSubpackages() || len(opts.BuildFlags.Include) > 0)) || opts.Build.Download {
		state.OutputDownload = core.OriginalOutputDownload
	}

	state.SetIncludeAndExclude(opts.BuildFlags.Include, opts.BuildFlags.Exclude)
	if opts.BuildFlags.Arch.OS != "" {
		state.TargetArch = opts.BuildFlags.Arch
	}

	// Only one target that is _not_ named "all" or "..." is allowed with debug test.
	if state.DebugFailingTests && (len(targets) != 1 || (len(targets) == 1 && (targets[0].IsPseudoTarget()))) {
		log.Fatalf("-d/--debug flag can only be used with a single test target")
	}

	if opts.Run.InTempDir && opts.Run.WD != "" {
		log.Fatal("Can't use both --in_temp_dir and --wd at the same time")
	}

	runPlease(state, targets)
	if state.RemoteClient != nil && !opts.Run.Remote {
		defer state.RemoteClient.Disconnect()
	}
	failures, _, _ := state.Failures()
	return !failures, state
}

func runPlease(state *core.BuildState, targets []core.BuildLabel) {
	// Every plz instance gets a shared repo lock which provides the following:
	// 1) Multiple plz instances can run concurrently.
	// 2) If another process tries to obtain an exclusive repo lock, it will have to wait until any existing repo locks are released in other processes.
	// This is useful for things like when plz tries to download and update itself.
	// 3) A new plz process will have to wait to acquire its shared repo lock, if there's already an existing process with an exclusive repo lock.
	core.AcquireSharedRepoLock()
	defer core.ReleaseRepoLock() // We can safely release the lock at this stage.

	core.StoreCurrentOperation()
	core.CheckXattrsSupported(state)

	detailedTests := state.NeedTests && (opts.Test.Detailed || opts.Cover.Detailed ||
		(len(targets) == 1 && !targets[0].IsPseudoTarget() && targets[0] != core.BuildLabelStdin))
	streamTests := opts.Test.StreamResults || opts.Cover.StreamResults
	shell := opts.Build.Shell != "" || opts.Test.Shell != "" || opts.Cover.Shell != ""
	shellRun := opts.Build.Shell == "run" || opts.Test.Shell == "run" || opts.Cover.Shell == "run"
	pretty := prettyOutput(opts.OutputFlags.InteractiveOutput, opts.OutputFlags.PlainOutput || opts.BehaviorFlags.Debug, opts.OutputFlags.Verbosity) && state.NeedBuild && !streamTests
	state.Cache = cache.NewCache(state)

	// Run the display
	state.Results() // important this is called now, don't ask...
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		output.MonitorState(state, !pretty, detailedTests, streamTests, shell, shellRun, string(opts.OutputFlags.TraceFile))
		wg.Done()
	}()
	plz.Run(targets, opts.BuildFlags.PreTargets, state, config, state.TargetArch)
	wg.Wait()
}

// testTargets handles test targets which can be given in two formats; a list of targets or a single
// target with a list of trailing arguments.
// Alternatively they can be completely omitted in which case we test everything under the working dir.
// One can also pass a 'failed' flag which runs the failed tests from last time.
func testTargets(target core.BuildLabel, inputs TargetsOrArgs, failed bool, resultsFile cli.Filepath) ([]core.BuildLabel, []string) {
	if failed {
		return test.LoadPreviousFailures(string(resultsFile))
	} else if target.Name == "" {
		return core.InitialPackage(), nil
	}
	labels, args := inputs.SeparateUnannotated()
	return append([]core.BuildLabel{target}, labels...), args
}

type TargetOrArg struct {
	arg   string
	label core.AnnotatedOutputLabel
}

func (arg TargetOrArg) Complete(match string) []flags.Completion {
	if core.LooksLikeABuildLabel(match) {
		var l core.BuildLabel
		return l.Complete(match)
	}
	return nil
}

func (arg *TargetOrArg) UnmarshalFlag(value string) error {
	if core.LooksLikeABuildLabel(value) {
		return arg.label.UnmarshalFlag(value)
	}
	arg.arg = value
	return nil
}

type TargetsOrArgs []TargetOrArg

// Separate splits the targets & arguments into the labels (in both annotated & unannotated forms) and the arguments.
func (l TargetsOrArgs) Separate(requireOneLabel bool) (annotated []core.AnnotatedOutputLabel, unannotated []core.BuildLabel, args []string) {
	if requireOneLabel && l[0].arg != "" && l[0].arg != "-" {
		if err := l[0].label.UnmarshalFlag(l[0].arg); err != nil {
			log.Fatalf("First argument must be a build label: %s", l[0].arg)
		}
	}
	for _, arg := range l {
		if l, _ := arg.label.Label(); l.IsEmpty() {
			if arg.arg == "-" {
				labels := plz.ReadAndParseStdinLabels()
				unannotated = append(unannotated, labels...)
				for _, label := range labels {
					annotated = append(annotated, core.AnnotatedOutputLabel{BuildLabel: label})
				}
			} else {
				args = append(args, arg.arg)
			}
		} else {
			annotated = append(annotated, arg.label)
			unannotated = append(unannotated, arg.label.BuildLabel)
		}
	}
	return
}

// SeparateUnannotated splits the targets & arguments into two slices. Annotations aren't permitted.
func (l TargetsOrArgs) SeparateUnannotated() ([]core.BuildLabel, []string) {
	annotated, unannotated, args := l.Separate(false)
	for _, a := range annotated {
		if a.Annotation != "" {
			log.Fatalf("Invalid build label; annotations are not permitted here: %s", a)
		}
	}
	return unannotated, args
}

// readConfig reads the initial configuration files
func readConfig() *core.Configuration {
	cfg, err := core.ReadDefaultConfigFiles(fs.HostFS, opts.BuildFlags.Profile)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	} else if err := cfg.ApplyOverrides(opts.BuildFlags.Option); err != nil {
		log.Fatalf("Can't override requested config setting: %s", err)
	}
	if opts.BehaviorFlags.HTTPProxy != "" {
		cfg.Build.HTTPProxy = opts.BehaviorFlags.HTTPProxy
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
	if stat, _ := os.Stdin.Stat(); (stat.Mode()&os.ModeCharDevice) == 0 && !plz.ReadingStdin(targets) {
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

// readConfigAndSetRoot returns an error if we can't find a repo root
func readConfigAndSetRoot(forceUpdate bool) (*core.Configuration, error) {
	if core.FindRepoRoot() {
		return mustReadConfigAndSetRoot(forceUpdate), nil
	}
	return nil, fmt.Errorf("failed to locate repo root")
}

// mustReadConfigAndSetRoot reads the .plzconfig files and moves to the repo root.
func mustReadConfigAndSetRoot(forceUpdate bool) *core.Configuration {
	if opts.BuildFlags.RepoRoot == "" {
		log.Debug("Found repo root at %s", core.MustFindRepoRoot())
	} else {
		abs, err := filepath.Abs(string(opts.BuildFlags.RepoRoot))
		if err != nil {
			log.Fatalf("Cannot make --repo_root absolute: %s", err)
		}
		core.RepoRoot = abs
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
		if !filepath.IsAbs(string(opts.OutputFlags.LogFile)) {
			opts.OutputFlags.LogFile = cli.Filepath(filepath.Join(core.RepoRoot, string(opts.OutputFlags.LogFile)))
		}
		cli.InitFileLogging(string(opts.OutputFlags.LogFile), opts.OutputFlags.LogFileLevel, opts.OutputFlags.LogAppend)
	}
	if opts.BehaviorFlags.NoHashVerification {
		log.Warning("You've disabled hash verification; this is intended to help temporarily while modifying build targets. You shouldn't use this regularly.")
	}
	config := readConfig()
	// Now apply any flags that override this
	if opts.Update.Latest || opts.Update.LatestPrerelease {
		config.Please.Version.Unset()
	} else if opts.Update.Version.IsSet {
		config.Please.Version = opts.Update.Version
	}
	update.CheckAndUpdate(config, !opts.BehaviorFlags.NoUpdate, forceUpdate, opts.Update.Force, !opts.Update.NoVerify, !opts.OutputFlags.PlainOutput, opts.Update.LatestPrerelease)
	return config
}

// handleCompletions handles shell completion. Typically it just prints to stdout but
// may do a little more if we think we need to handle aliases.
func handleCompletions(parser *flags.Parser, items []flags.Completion) {
	cli.InitLogging(cli.MinVerbosity)  // Ensure this is quiet
	opts.BehaviorFlags.NoUpdate = true // Ensure we don't try to update
	if len(items) > 0 && items[0].Description == "BuildLabel" {
		// Don't muck around with the config if we're predicting build labels.
		cli.PrintCompletions(items)
	} else if config := mustReadConfigAndSetRoot(false); config.AttachAliasFlags(parser) {
		// Run again without this registered as a completion handler
		parser.CompletionHandler = nil
		parser.ParseArgs(os.Args[1:])
	} else {
		cli.PrintCompletions(items)
	}
	// Regardless of what happened, always exit with 0 at this point.
	os.Exit(0)
}

// Capture aliases from config file and print to the help output
func additionalUsageInfo(parser *flags.Parser, wr io.Writer) {
	cli.InitLogging(cli.MinVerbosity)
	if config, err := readConfigAndSetRoot(false); err == nil && parser.Active == nil {
		config.PrintAliases(wr)
	}
}

func getCompletions(qry string) (*query.CompletionPackages, []string) {
	binary := opts.Query.Completions.Cmd == "run" || opts.Query.Completions.Cmd == "exec"
	isTest := opts.Query.Completions.Cmd == "test" || opts.Query.Completions.Cmd == "cover"

	completions := query.CompletePackages(config, qry)

	if completions.PackageToParse != "" || completions.IsRoot {
		labelsToParse := []core.BuildLabel{{PackageName: completions.PackageToParse, Name: "all"}}
		if success, state := Please(labelsToParse, config, false, false); success {
			return completions, query.Completions(state.Graph, completions, binary, isTest, completions.Hidden)
		}
	}
	return completions, nil
}

func initBuild(args []string) string {
	if len(args) > 1 && (args[1] == "sandbox") {
		// Shortcut these as they're special commands used for please sandboxing
		// going through the normal init path would be too slow
		return args[1]
	}
	if _, present := os.LookupEnv("GO_FLAGS_COMPLETION"); present {
		cli.InitLogging(cli.MinVerbosity)
	}
	parser, extraArgs, flagsErr := cli.ParseFlags("Please", &opts, args, flags.PassDoubleDash, handleCompletions, additionalUsageInfo)
	// Note that we must leave flagsErr for later, because it may be affected by aliases.
	if opts.HelpFlags.Version {
		fmt.Printf("Please version %s\n", core.PleaseVersion)
		os.Exit(0) // Ignore other flags if --version was passed.
	} else if opts.HelpFlags.Help {
		parser.WriteHelp(os.Stderr)
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
	if _, present := os.LookupEnv("SUDO_COMMAND"); present && !opts.BehaviorFlags.AllowSudo {
		log.Fatalf("Refusing to run under sudo; generally it is a very bad idea to invoke Please in this way. You can pass --allow_sudo to permit it, but almost certainly you do not want to do this.")
	}
	if _, err := maxprocs.Set(maxprocs.Logger(log.Info), maxprocs.Min(opts.BuildFlags.NumThreads)); err != nil {
		log.Error("Failed to set GOMAXPROCS: %s", err)
	}

	command := cli.ActiveFullCommand(parser.Command)
	if opts.Complete != "" {
		// Completion via PLZ_COMPLETE env var sidesteps other commands
		opts.Query.Completions.Cmd = command
		opts.Query.Completions.Args.Fragments = []string{opts.Complete}
		command = "query.completions"
	} else if command == "help" || command == "init" || command == "init.config" || command == "tool" {
		// These commands don't use a config file, allowing them to be run outside a repo.
		if flagsErr != nil { // This error otherwise doesn't get checked until later.
			cli.ParseFlagsFromArgsOrDie("Please", &opts, os.Args, additionalUsageInfo)
		}
		config = core.DefaultConfiguration()
		os.Exit(buildFunctions[command]())
	} else if opts.OutputFlags.CompletionScript {
		fmt.Printf("%s\n", string(assets.PlzComplete))
		os.Exit(0)
	} else if opts.Clean.Rm != "" {
		// Avoid initialising logging so we don't create an additional file.
		if err := fs.RemoveAll(opts.Clean.Rm); err != nil {
			log.Fatalf("%s", err)
		}
		os.Exit(0)
	}
	// Read the config now
	config = mustReadConfigAndSetRoot(command == "update")
	if parser.Command.Active != nil && parser.Command.Active.Name == "query" {
		// Query commands don't need either of these set.
		opts.OutputFlags.PlainOutput = true
		config.Cache.DirClean = false
	}

	// Now we've read the config file, we may need to re-run the parser; the aliases in the config
	// can affect how we parse otherwise illegal flag combinations.
	if (flagsErr != nil || len(extraArgs) > 0) && command != "query.completions" {
		args := config.UpdateArgsWithAliases(os.Args)
		parser, _, err := cli.ParseFlags("Please", &opts, args, flags.PassDoubleDash, handleCompletions, additionalUsageInfo)
		if err != nil {
			log.Fatalf("%s", err)
		}
		command = cli.ActiveFullCommand(parser.Command)
	}

	if opts.ProfilePort != 0 {
		go func() {
			log.Warning("%s", http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", opts.ProfilePort), nil))
		}()
	}
	return command
}

func writeGoTraceFile() {
	if err := runtime.StartTrace(); err != nil {
		log.Fatalf("failed to start trace: %v", err)
	}

	f, err := os.Create(opts.GoTraceFile)
	if err != nil {
		log.Fatalf("Failed to create trace file: %v", err)
	}
	defer f.Close()

	for {
		data := runtime.ReadTrace()
		if data == nil {
			return
		}
		if _, err := f.Write(data); err != nil {
			log.Fatalf("Failed to write trace data: %v", err)
		}
	}
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
	} else if _, buildFailed, testFailed := state.Failures(); buildFailed {
		return 2
	} else if testFailed {
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
	if opts.MutexProfile != "" {
		runtime.SetMutexProfileFraction(1)
		f, err := os.Create(opts.MutexProfile)
		if err != nil {
			log.Fatalf("Failed to open mutex profile file: %s", err)
		}
		defer f.Close()
		defer func() {
			pprof.Lookup("mutex").WriteTo(f, 0)
		}()
	}
	if opts.GoTraceFile != "" {
		go writeGoTraceFile()
		defer func() {
			runtime.StopTrace()
		}()
	}

	log.Debugf("plz %v", command)
	return buildFunctions[command]()
}

func main() {
	os.Exit(execute(initBuild(os.Args)))
}
