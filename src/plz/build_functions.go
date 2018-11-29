package plz

//
//import (
//	"fmt"
//	"net/http"
//	_ "net/http/pprof"
//	"os"
//	"path"
//	"runtime"
//	"runtime/pprof"
//	"strings"
//	"syscall"
//	"time"
//
//	"github.com/jessevdk/go-flags"
//	"gopkg.in/op/go-logging.v1"
//
//	"build"
//	"cache"
//	"clean"
//	"cli"
//	"core"
//	"export"
//	"follow"
//	"fs"
//	"gc"
//	"hashes"
//	"help"
//	"ide/intellij"
//	"metrics"
//	"output"
//	"parse"
//	"query"
//	"run"
//	"sync"
//	"test"
//	"tool"
//	"update"
//	"utils"
//	"watch"
//	"worker"
//)
//
//var log = logging.MustGetLogger("plz")
//
//var config *core.Configuration
//
//// Definitions of what we do for each command.
//// Functions are called after args are parsed and return true for success.
//var buildFunctions = map[string]func() bool{
//	"build": func() bool {
//		success, _ := runBuild(opts.Build.Args.Targets, true, false)
//		return success
//	},
//	"rebuild": func() bool {
//		// It would be more pure to require --nocache for this, but in basically any context that
//		// you use 'plz rebuild', you don't want the cache coming in and mucking things up.
//		// 'plz clean' followed by 'plz build' would still work in those cases, anyway.
//		opts.FeatureFlags.NoCache = true
//		success, _ := runBuild(opts.Rebuild.Args.Targets, true, false)
//		return success
//	},
//	"hash": func() bool {
//		success, state := runBuild(opts.Hash.Args.Targets, true, false)
//		if opts.Hash.Detailed {
//			for _, target := range state.ExpandOriginalTargets() {
//				build.PrintHashes(state, state.Graph.TargetOrDie(target))
//			}
//		}
//		if opts.Hash.Update {
//			hashes.RewriteHashes(state, state.ExpandOriginalTargets())
//		}
//		return success
//	},
//	"test": func() bool {
//		targets := testTargets(opts.Test.Args.Target, opts.Test.Args.Args, opts.Test.Failed, opts.Test.TestResultsFile)
//		success, _ := doTest(targets, opts.Test.SurefireDir, opts.Test.TestResultsFile)
//		return success || opts.Test.FailingTestsOk
//	},
//	"cover": func() bool {
//		if opts.BuildFlags.Config != "" {
//			log.Warning("Build config overridden; coverage may not be available for some languages")
//		} else {
//			opts.BuildFlags.Config = "cover"
//		}
//		targets := testTargets(opts.Cover.Args.Target, opts.Cover.Args.Args, opts.Cover.Failed, opts.Cover.TestResultsFile)
//		os.RemoveAll(string(opts.Cover.CoverageResultsFile))
//		success, state := doTest(targets, opts.Cover.SurefireDir, opts.Cover.TestResultsFile)
//		test.AddOriginalTargetsToCoverage(state, opts.Cover.IncludeAllFiles)
//		test.RemoveFilesFromCoverage(state.Coverage, state.Config.Cover.ExcludeExtension)
//
//		test.WriteCoverageToFileOrDie(state.Coverage, string(opts.Cover.CoverageResultsFile))
//		test.WriteXMLCoverageToFileOrDie(targets, state.Coverage, string(opts.Cover.CoverageXMLReport))
//
//		if opts.Cover.LineCoverageReport {
//			output.PrintLineCoverageReport(state, opts.Cover.IncludeFile)
//		} else if !opts.Cover.NoCoverageReport {
//			output.PrintCoverage(state, opts.Cover.IncludeFile)
//		}
//		return success || opts.Cover.FailingTestsOk
//	},
//	"run": func() bool {
//		if success, state := runBuild([]core.BuildLabel{opts.Run.Args.Target}, true, false); success {
//			run.Run(state, opts.Run.Args.Target, opts.Run.Args.Args, opts.Run.Env)
//		}
//		return false // We should never return from run.Run so if we make it here something's wrong.
//	},
//	"parallel": func() bool {
//		if success, state := runBuild(opts.Run.Parallel.PositionalArgs.Targets, true, false); success {
//			if opts.Watch.Run {
//				run.Parallel(state, state.ExpandOriginalTargets(), opts.Run.Parallel.Args, opts.Run.Parallel.NumTasks, opts.Run.Parallel.Quiet, opts.Run.Env)
//			} else {
//				os.Exit(run.Parallel(state, state.ExpandOriginalTargets(), opts.Run.Parallel.Args, opts.Run.Parallel.NumTasks, opts.Run.Parallel.Quiet, opts.Run.Env))
//			}
//		}
//		return false
//	},
//	"sequential": func() bool {
//		if success, state := runBuild(opts.Run.Sequential.PositionalArgs.Targets, true, false); success {
//			os.Exit(run.Sequential(state, state.ExpandOriginalTargets(), opts.Run.Sequential.Args, opts.Run.Sequential.Quiet, opts.Run.Env))
//		}
//		return false
//	},
//	"clean": func() bool {
//		config.Cache.DirClean = false
//		if len(opts.Clean.Args.Targets) == 0 {
//			if len(opts.BuildFlags.Include) == 0 && len(opts.BuildFlags.Exclude) == 0 {
//				// Clean everything, doesn't require parsing at all.
//				if !opts.Clean.Remote {
//					// Don't construct the remote caches if they didn't pass --remote.
//					config.Cache.RPCURL = ""
//					config.Cache.HTTPURL = ""
//				}
//				clean.Clean(config, newCache(config), !opts.Clean.NoBackground)
//				return true
//			}
//			opts.Clean.Args.Targets = core.WholeGraph
//		}
//		if success, state := runBuild(opts.Clean.Args.Targets, false, false); success {
//			clean.Targets(state, state.ExpandOriginalTargets(), !opts.FeatureFlags.NoCache)
//			return true
//		}
//		return false
//	},
//	"update": func() bool {
//		fmt.Printf("Up to date (version %s).\n", core.PleaseVersion)
//		return true // We'd have died already if something was wrong.
//	},
//	"op": func() bool {
//		cmd := core.ReadLastOperationOrDie()
//		log.Notice("OP PLZ: %s", strings.Join(cmd, " "))
//		// Annoyingly we don't seem to have any access to execvp() which would be rather useful here...
//		executable, err := os.Executable()
//		if err == nil {
//			err = syscall.Exec(executable, append([]string{executable}, cmd...), os.Environ())
//		}
//		log.Fatalf("SORRY OP: %s", err) // On success Exec never returns.
//		return false
//	},
//	"gc": func() bool {
//		success, state := runBuild(core.WholeGraph, false, false)
//		if success {
//			state.OriginalTargets = state.Config.Gc.Keep
//			gc.GarbageCollect(state, opts.Gc.Args.Targets, state.ExpandOriginalTargets(), state.Config.Gc.Keep, state.Config.Gc.KeepLabel,
//				opts.Gc.Conservative, opts.Gc.TargetsOnly, opts.Gc.SrcsOnly, opts.Gc.NoPrompt, opts.Gc.DryRun, opts.Gc.Git)
//		}
//		return success
//	},
//	"init": func() bool {
//		utils.InitConfig(string(opts.Init.Dir), opts.Init.BazelCompatibility)
//		return true
//	},
//	"export": func() bool {
//		success, state := runBuild(opts.Export.Args.Targets, false, false)
//		if success {
//			export.ToDir(state, opts.Export.Output, state.ExpandOriginalTargets())
//		}
//		return success
//	},
//	"follow": func() bool {
//		// This is only temporary, ConnectClient will alter it to match the server.
//		state := core.NewBuildState(1, nil, int(opts.OutputFlags.Verbosity), config)
//		return follow.ConnectClient(state, opts.Follow.Args.URL.String(), opts.Follow.Retries, time.Duration(opts.Follow.Delay))
//	},
//	"outputs": func() bool {
//		success, state := runBuild(opts.Export.Outputs.Args.Targets, true, false)
//		if success {
//			export.Outputs(state, opts.Export.Output, state.ExpandOriginalTargets())
//		}
//		return success
//	},
//	"help": func() bool {
//		return help.Help(string(opts.Help.Args.Topic))
//	},
//	"tool": func() bool {
//		tool.Run(config, opts.Tool.Args.Tool, opts.Tool.Args.Args)
//		return false // If the function returns (which it shouldn't), something went wrong.
//	},
//	"deps": func() bool {
//		return runQuery(true, opts.Query.Deps.Args.Targets, func(state *core.BuildState) {
//			query.Deps(state, state.ExpandOriginalTargets(), opts.Query.Deps.Unique, opts.Query.Deps.Level)
//		})
//	},
//	"reverseDeps": func() bool {
//		opts.VisibilityParse = true
//		return runQuery(false, opts.Query.ReverseDeps.Args.Targets, func(state *core.BuildState) {
//			query.ReverseDeps(state, state.ExpandOriginalTargets())
//		})
//	},
//	"somepath": func() bool {
//		return runQuery(true,
//			[]core.BuildLabel{opts.Query.SomePath.Args.Target1, opts.Query.SomePath.Args.Target2},
//			func(state *core.BuildState) {
//				query.SomePath(state.Graph, opts.Query.SomePath.Args.Target1, opts.Query.SomePath.Args.Target2)
//			},
//		)
//	},
//	"alltargets": func() bool {
//		return runQuery(true, opts.Query.AllTargets.Args.Targets, func(state *core.BuildState) {
//			query.AllTargets(state.Graph, state.ExpandOriginalTargets(), opts.Query.AllTargets.Hidden)
//		})
//	},
//	"print": func() bool {
//		return runQuery(false, opts.Query.Print.Args.Targets, func(state *core.BuildState) {
//			query.Print(state.Graph, state.ExpandOriginalTargets(), opts.Query.Print.Fields)
//		})
//	},
//	"affectedtargets": func() bool {
//		files := opts.Query.AffectedTargets.Args.Files
//		targets := core.WholeGraph
//		if opts.Query.AffectedTargets.Intransitive {
//			state := core.NewBuildState(1, nil, 1, config)
//			targets = core.FindOwningPackages(state, files)
//		}
//		return runQuery(true, targets, func(state *core.BuildState) {
//			query.AffectedTargets(state, files.Get(), opts.BuildFlags.Include, opts.BuildFlags.Exclude, opts.Query.AffectedTargets.Tests, !opts.Query.AffectedTargets.Intransitive)
//		})
//	},
//	"input": func() bool {
//		return runQuery(true, opts.Query.Input.Args.Targets, func(state *core.BuildState) {
//			query.TargetInputs(state.Graph, state.ExpandOriginalTargets())
//		})
//	},
//	"output": func() bool {
//		return runQuery(true, opts.Query.Output.Args.Targets, func(state *core.BuildState) {
//			query.TargetOutputs(state.Graph, state.ExpandOriginalTargets())
//		})
//	},
//	"completions": func() bool {
//		// Somewhat fiddly because the inputs are not necessarily well-formed at this point.
//		opts.ParsePackageOnly = true
//		fragments := opts.Query.Completions.Args.Fragments.Get()
//		if opts.Query.Completions.Cmd == "help" {
//			// Special-case completing help topics rather than build targets.
//			if len(fragments) == 0 {
//				help.Topics("")
//			} else {
//				help.Topics(fragments[0])
//			}
//			return true
//		}
//		if len(fragments) == 0 || len(fragments) == 1 && strings.Trim(fragments[0], "/ ") == "" {
//			os.Exit(0) // Don't do anything for empty completion, it's normally too slow.
//		}
//		labels, parseLabels, hidden := query.CompletionLabels(config, fragments, core.RepoRoot)
//		if success, state := Please(parseLabels, config, false, false, false); success {
//			binary := opts.Query.Completions.Cmd == "run"
//			test := opts.Query.Completions.Cmd == "test" || opts.Query.Completions.Cmd == "cover"
//			query.Completions(state.Graph, labels, binary, test, hidden)
//			return true
//		}
//		return false
//	},
//	"graph": func() bool {
//		return runQuery(true, opts.Query.Graph.Args.Targets, func(state *core.BuildState) {
//			if len(opts.Query.Graph.Args.Targets) == 0 {
//				state.OriginalTargets = opts.Query.Graph.Args.Targets // It special-cases doing the full graph.
//			}
//			query.Graph(state, state.ExpandOriginalTargets())
//		})
//	},
//	"whatoutputs": func() bool {
//		return runQuery(true, core.WholeGraph, func(state *core.BuildState) {
//			query.WhatOutputs(state.Graph, opts.Query.WhatOutputs.Args.Files.Get(), opts.Query.WhatOutputs.EchoFiles)
//		})
//	},
//	"rules": func() bool {
//		success, state := Please(opts.Query.Rules.Args.Targets, config, true, true, false)
//		if success {
//			parse.PrintRuleArgs(state, state.ExpandOriginalTargets())
//		}
//		return success
//	},
//	"changed": func() bool {
//		success, state := runBuild(core.WholeGraph, false, false)
//		if !success {
//			return false
//		}
//		for _, label := range query.ChangedLabels(
//			state,
//			query.ChangedRequest{
//				Since:            opts.Query.Changed.Since,
//				DiffSpec:         opts.Query.Changed.DiffSpec,
//				IncludeDependees: opts.Query.Changed.IncludeDependees,
//			}) {
//			fmt.Printf("%s\n", label)
//		}
//		return true
//	},
//	"changes": func() bool {
//		// Temporarily set this flag on to avoid fatal errors from the first parse.
//		keepGoing := opts.BuildFlags.KeepGoing
//		opts.BuildFlags.KeepGoing = true
//		original := query.MustGetRevision(opts.Query.Changes.CurrentCommand)
//		files := opts.Query.Changes.Args.Files.Get()
//		query.MustCheckout(opts.Query.Changes.Since, opts.Query.Changes.CheckoutCommand)
//		_, before := runBuild(core.WholeGraph, false, false)
//		opts.BuildFlags.KeepGoing = keepGoing
//		// N.B. Ignore failure here; if we can't parse the graph before then it will suffice to
//		//      assume that anything we don't know about has changed.
//		query.MustCheckout(original, opts.Query.Changes.CheckoutCommand)
//		success, after := runBuild(core.WholeGraph, false, false)
//		if !success {
//			return false
//		}
//		for _, target := range query.DiffGraphs(before, after, files) {
//			fmt.Printf("%s\n", target)
//		}
//		return true
//	},
//	"roots": func() bool {
//		return runQuery(true, opts.Query.Roots.Args.Targets, func(state *core.BuildState) {
//			query.Roots(state.Graph, opts.Query.Roots.Args.Targets)
//		})
//	},
//	"watch": func() bool {
//		opts.Watch.Watching = true
//		success, state := runBuild(opts.Watch.Args.Targets, true, true)
//		watchedProcessName := setWatchedTarget(state, state.ExpandOriginalTargets())
//		watch.Watch(state, state.ExpandOriginalTargets(), watchedProcessName, runWatchedBuild)
//		return success
//	},
//	"filter": func() bool {
//		return runQuery(false, opts.Query.Filter.Args.Targets, func(state *core.BuildState) {
//			query.Filter(state, state.ExpandOriginalTargets())
//		})
//	},
//	"intellij": func() bool {
//		success, state := runBuild(opts.Ide.IntelliJ.Args.Labels, false, false)
//		if success {
//			intellij.ExportIntellijStructure(state.Config, state.Graph, state.ExpandOriginalLabels())
//		}
//		return success
//	},
//}
