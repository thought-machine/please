package core

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/fs"
)

// OutDir is the root output directory for everything.
const OutDir = "plz-out"

// TmpDir is the root of the temporary directory for building targets & running tests.
const TmpDir = "plz-out/tmp"

// GenDir is the output directory for non-binary targets.
const GenDir = "plz-out/gen"

// BinDir is the output directory for binary targets.
const BinDir = "plz-out/bin"

// ExecDir is the output directory that we execute in.
const ExecDir = "plz-out/exec"

// SubrepoDir is the output directory for targets that define subrepos.
const SubrepoDir = "plz-out/subrepos"

// DefaultBuildingDescription is the default description for targets when they're building.
const DefaultBuildingDescription = "Building..."

// SandboxDir is the directory that sandboxed actions are run in.
const SandboxDir = "/tmp/plz_sandbox"

// Suffixes for temporary directories
const buildDirSuffix = "._build"
const testDirSuffix = "._test"

// TestResultsFile is the file that targets output their test results into.
// This is normally defined for them via an environment variable.
const TestResultsFile = "test.results"

// CoverageFile is the file that targets output coverage information into.
// This is similarly defined via an environment variable.
const CoverageFile = "test.coverage"

// tempOutputSuffix is the suffix we attach to temporary outputs to avoid name clashes.
const tempOutputSuffix = ".out"

// lockFileSuffix is the suffix we attach to lock files.
const lockFileSuffix = ".lock"

type TestFields struct {
	// Shell command to run for test targets.
	Command string `name:"test_cmd"`
	// Per-configuration test commands to run.
	Commands map[string]string `name:"test_cmd"`
	// The results of this test target, if it is one.
	Results *TestSuite `print:"false"`
	// Like tools but available to the test_cmd instead
	tools []BuildInput `name:"test_tools"`
	// Named test tools, similar to named sources.
	namedTools map[string][]BuildInput `name:"test_tools"`
	// The timeout for the test
	Timeout time.Duration `name:"test_timeout"`
	// Extra output files from the test.
	// These are in addition to the usual test.results output file.
	Outputs []string `name:"test_outputs"`
	// Flakiness of test, ie. number of times we will rerun it before giving up. 1 is the default.
	Flakiness uint8 `name:"flaky"`
	// True if the test action is sandboxed.
	Sandbox bool `name:"test_sandbox"`
	// True if the target is a test and has no output file.
	// Default is false, meaning all tests must produce test.results as output.
	NoOutput bool `name:"no_test_output"`
}

type DebugFields struct {
	// Shell command to debug this rule.
	Command string `name:"debug_cmd"`
	// Debug data files of this rule.
	data []BuildInput `name:"debug_data"`
	// Debug data files of this rule by name.
	namedData map[string][]BuildInput `name:"debug_data"`
	// Tools available to the debug_cmd.
	tools []BuildInput `name:"debug_tools"`
	// Tools available to the debug_cmd by name.
	namedTools map[string][]BuildInput `name:"debug_tools"`
}

// A BuildTarget is a representation of a build target and all information about it;
// its name, dependencies, build commands, etc.
type BuildTarget struct {
	// N.B. The tags on these fields are used by query print to help it print them.

	// Identifier of this build target
	Label BuildLabel `name:"name"`
	// If this target is in a subrepo, this will be the one it's in.
	Subrepo *Subrepo `print:"false"`
	// Dependencies of this target.
	// Maps the original declaration to whatever dependencies actually got attached,
	// which may be more than one in some cases. Also contains info about exporting etc.
	dependencies []depInfo `name:"deps"`
	// List of build target patterns that can use this build target.
	Visibility []BuildLabel
	// Source files of this rule. Can refer to build rules themselves.
	Sources []BuildInput `name:"srcs"`
	// Named source files of this rule; as above but identified by name.
	NamedSources map[string][]BuildInput `name:"srcs"`
	// Data files of this rule. Similar to sources but used at runtime, typically by tests.
	Data []BuildInput `name:"data"`
	// Data files of this rule by name.
	NamedData map[string][]BuildInput `name:"data"`
	// Output files of this rule. All are paths relative to this package.
	outputs []string `name:"outs" hide:"filegroup"`
	// Named output subsets of this rule. All are paths relative to this package but can be
	// captured separately; for example something producing C code might separate its outputs
	// into sources and headers.
	namedOutputs map[string][]string `name:"outs" hide:"filegroup"`
	// Optional output files of this rule. Same as outs but aren't required to be produced always.
	// Can be glob patterns.
	OptionalOutputs []string `name:"optional_outs"`
	// Optional labels applied to this rule. Used for including/excluding rules.
	Labels []string
	// Shell command to run.
	Command string `name:"cmd" hide:"filegroup"`
	// Per-configuration shell commands to run.
	Commands map[string]string `name:"cmd" hide:"filegroup"`
	// Test related fields.
	Test *TestFields `name:"test"`
	// Debug related fields.
	Debug *DebugFields
	// If ShowProgress is true, this is used to store the current progress of the target.
	Progress atomicFloat32 `print:"false"`
	// For remote_files, this is the total size of the download (if known)
	FileSize uint64 `print:"false"`
	// Description displayed while the command is building.
	// Default is just "Building" but it can be customised.
	BuildingDescription string `name:"building_description"`
	// Acceptable hashes of the outputs of this rule. If the output doesn't match any of these
	// it's an error at build time. Can be used to validate third-party deps.
	Hashes []string
	// Licences that this target is subject to.
	Licences []string
	// Any secrets that this rule requires.
	// Secrets are similar to sources but are always absolute system paths and affect the hash
	// differently; they are not used to determine the hash for retrieving a file from cache, but
	// if changed locally will still force a rebuild. They're not copied into the source directory
	// (or indeed anywhere by plz).
	Secrets []string
	// Named secrets of this rule; as above but identified by name.
	NamedSecrets map[string][]string
	// BUILD language functions to call before / after target is built. Allows deferred manipulation of the build graph.
	PreBuildFunction  PreBuildFunction  `name:"pre_build"`
	PostBuildFunction PostBuildFunction `name:"post_build"`
	// Languages this rule requires. These are an arbitrary set and the only meaning is that they
	// correspond to entries in Provides; if rules match up then it allows choosing a specific
	// dependency (consider eg. code generated from protobufs; this mechanism allows us to expose
	// one rule but only compile the appropriate code for each library that consumes it).
	Requires []string
	// Dependent rules this rule provides for each language. Matches up to Requires as described above.
	Provides map[string][]BuildLabel
	// Stores the hash of this build rule before any post-build function is run.
	RuleHash []byte `name:"exported_deps"` // bit of a hack to call this exported_deps...
	// Tools that this rule will use, ie. other rules that it may use at build time which are not
	// copied into its source directory.
	Tools []BuildInput
	// Named tools, similar to named sources.
	namedTools map[string][]BuildInput `name:"tools"`
	// Target-specific environment passthroughs.
	PassEnv *[]string `name:"pass_env"`
	// Target-specific unsafe environment passthroughs.
	PassUnsafeEnv *[]string `name:"pass_unsafe_env"`

	// Timeouts for build/test actions
	BuildTimeout time.Duration `name:"build_timeout"`
	// OutputDirectories are the directories that outputs can be produced into which will be added to the root of the
	// output for the rule. For example if an output directory "foo" contains "bar.txt" the rule will have the output
	// "bar.txt"
	OutputDirectories []OutputDirectory `name:"output_dirs"`
	// EntryPoints represent named binaries within the rules output that can be targeted via //package:rule|entry_point_name
	EntryPoints map[string]string `name:"entry_points"`
	// Used to arbitrate concurrent access to dependencies, and to the test results.
	mutex sync.RWMutex `print:"false"`
	// Used to notify once this target has built successfully.
	finishedBuilding chan struct{} `print:"false"`
	// Env are any custom environment variables to set for this build target
	Env map[string]string `name:"env"`
	// The content of text_file() rules
	FileContent string `name:"content"`
	// Represents the state of this build target (see below)
	state int32 `print:"false"`
	// If true, the target is needed for a subinclude and therefore we will have to make sure its
	// outputs are available locally when built.
	neededForSubinclude atomic.Bool `print:"false"`
	// The number of completed runs
	completedRuns uint16 `print:"false"`
	// True if this target is a binary (ie. runnable, will appear in plz-out/bin)
	IsBinary bool `name:"binary"`
	// True if this target is an input for a subrepo; if so outputs will appear in plz-out/sub.
	IsSubrepo bool `name:"subrepo"`
	// Indicates that the target can only be depended on by tests or other rules with this set.
	// Used to restrict non-deployable code and also affects coverage detection.
	TestOnly bool `name:"test_only"`
	// True if the build action is sandboxed.
	Sandbox bool
	// True if this target needs access to its transitive dependencies to build.
	// This would be false for most 'normal' genrules but true for eg. compiler steps
	// that need to build in everything.
	NeedsTransitiveDependencies bool `name:"needs_transitive_deps"`
	// True if this target blocks recursive exploring for transitive dependencies.
	// This is typically false for _library rules which aren't complete, and true
	// for _binary rules which normally are, and genrules where you don't care about
	// the inputs, only whatever they were turned into.
	OutputIsComplete bool `name:"output_is_complete"`
	// If true, the rule is given an env var at build time that contains the hash of its
	// transitive dependencies, which can be used to identify the output in a predictable way.
	Stamp bool
	// If true, the target must be run locally (i.e. is not compatible with remote execution).
	Local bool
	// If true, the executed commands will exit whenever an error is encountered (i.e. shells
	// are executed with -e).
	ExitOnError bool
	// Marks the target as a filegroup.
	IsFilegroup bool `print:"false"`
	// Marks the target as a remote_file.
	IsRemoteFile bool `print:"false"`
	// Marks the target as a text_file.
	IsTextFile bool `print:"false"`
	// Marks that the target was added in a post-build function.
	AddedPostBuild bool `print:"false"`
	// If true, the interactive progress display will try to infer the target's progress
	// via some heuristics on its output.
	showProgress atomic.Bool `name:"progress"`
}

// ExpectedBuildMetadataVersionTag is the version tag that the current Please version expects. If this doesn't match
// the VersionTag of the BuildMetadata object, then Please will not use the cached target metadata. Changing this value
// will invalidate the local action metadata cache, which can be useful if it has been poisoned by a bug in a
// previous version of Please.
var ExpectedBuildMetadataVersionTag = 1

// BuildMetadata is temporary metadata that's stored around a build target - we don't
// generally persist it indefinitely.
type BuildMetadata struct {
	// Standard output & error
	Stdout, Stderr []byte
	// Serialised build action metadata.
	RemoteAction  []byte
	RemoteOutputs []byte
	// Time this action was written. Used for remote execution to determine if
	// the action is stale and needs re-checking or not.
	Timestamp time.Time
	// Additional optional outputs found from wildcard
	OptionalOutputs []string
	// Additional outputs from output directories serialised as a csv
	OutputDirOuts []string
	// True if this represents a test run.
	Test bool
	// True if the results were retrieved from a cache, false if we ran the full build action.
	Cached bool
	// VersionTag is an integer representing the version of this cache object. If this doesn't match the
	// expected version above, Please will not use this cached metadata.
	VersionTag int
}

// A PreBuildFunction is a type that allows hooking a pre-build callback.
type PreBuildFunction interface {
	fmt.Stringer
	// Call calls this pre-build function
	Call(target *BuildTarget) error
}

// A PostBuildFunction is a type that allows hooking a post-build callback.
type PostBuildFunction interface {
	fmt.Stringer
	// Call calls this pre-build function with this target and its output.
	Call(target *BuildTarget, output string) error
}

type depInfo struct {
	declared *BuildLabel    // the originally declared dependency
	deps     []*BuildTarget // list of actual deps
	resolved bool           // has the graph resolved it
	exported bool           // is it an exported dependency
	internal bool           // is it an internal dependency (that is not picked up implicitly by transitive searches)
	source   bool           // is it implicit because it's a source (not true if it's a dependency too)
	data     bool           // is it a data item for a test
}

// OutputDirectory is an output directory for the build rule. It may have a suffix of /** which means that we should
// traverse the directory tree adding each file individually rather than just adding whatever files/directories are in
// the top level.
type OutputDirectory string

// Dir returns the actual directory name for this output directory
func (o OutputDirectory) Dir() string {
	return strings.TrimSuffix(string(o), "/**")
}

// ShouldAddFiles checks whether the contents of this directory should include all the files in the directory tree
// individually i.e. out_dir/net/thoughtmachine/Main.java -> net/thoughtmachine/Main.java. If this is false then these
// files would be included as out_dir/net/thoughtmachine/Main.java -> net.
func (o OutputDirectory) ShouldAddFiles() bool {
	return strings.HasSuffix(string(o), "/**")
}

// A BuildTargetState tracks the current state of this target in regard to whether it's built
// or not. Targets only move forwards through this (i.e. the state of a target only ever increases).
type BuildTargetState uint8

// The available states for a target.
const (
	Inactive         BuildTargetState = iota // Target isn't used in current build
	Semiactive                               // Target would be active if we needed a build
	Active                                   // Target is going to be used in current build
	Pending                                  // Target is ready to be built but not yet started.
	Building                                 // Target is currently being built
	Stopped                                  // We stopped building the target because we'd gone as far as needed.
	Built                                    // Target has been successfully built
	Cached                                   // Target has been retrieved from the cache
	Unchanged                                // Target has been built but hasn't changed since last build
	Reused                                   // Outputs of previous build have been reused.
	BuiltRemotely                            // Target has been built but outputs are not necessarily local.
	ReusedRemotely                           // Outputs of previous remote action have been reused.
	DependencyFailed                         // At least one dependency of this target has failed.
	Failed                                   // Target failed for some reason
)

// String implements the fmt.Stringer interface.
func (s BuildTargetState) String() string {
	if s == Inactive {
		return "Inactive"
	} else if s == Semiactive {
		return "Semiactive"
	} else if s == Active {
		return "Active"
	} else if s == Pending {
		return "Pending"
	} else if s == Building {
		return "Building"
	} else if s == Stopped {
		return "Stopped"
	} else if s == Built {
		return "Built"
	} else if s == Cached {
		return "Cached"
	} else if s == Unchanged {
		return "Unchanged"
	} else if s == Reused {
		return "Reused"
	} else if s == DependencyFailed {
		return "Dependency Failed"
	} else if s == Failed {
		return "Failed"
	} else if s == BuiltRemotely {
		return "Built remotely"
	} else if s == ReusedRemotely {
		return "Reused remote outputs"
	}
	return "Unknown"
}

func (s BuildTargetState) IsBuilt() bool {
	return Built <= s && s < DependencyFailed
}

// NewBuildTarget constructs & returns a new BuildTarget.
func NewBuildTarget(label BuildLabel) *BuildTarget {
	return &BuildTarget{
		Label:               label,
		state:               int32(Inactive),
		BuildingDescription: DefaultBuildingDescription,
		finishedBuilding:    make(chan struct{}),
	}
}

// String returns a stringified form of the build label of this target, which is
// a unique identity for it.
func (target *BuildTarget) String() string {
	return target.Label.String()
}

// TmpDir returns the temporary working directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy._build
// Note the extra subdirectory to keep rules separate from one another, and the .build suffix
// to attempt to keep rules from duplicating the names of sub-packages; obviously that is not
// 100% reliable but we don't have a better solution right now.
func (target *BuildTarget) TmpDir() string {
	return filepath.Join(TmpDir, target.Label.Subrepo, target.Label.PackageName, target.Label.Name+buildDirSuffix)
}

// BuildLockFile returns the lock filename for the target's build stage.
func (target *BuildTarget) BuildLockFile() string {
	return target.TmpDir() + lockFileSuffix
}

// OutDir returns the output directory for this target, eg.
// //mickey/donald:goofy -> plz-out/gen/mickey/donald (or plz-out/bin if it's a binary)
func (target *BuildTarget) OutDir() string {
	if target.IsSubrepo {
		return filepath.Join(SubrepoDir, target.Label.Subrepo, target.Label.PackageName)
	} else if target.IsBinary {
		return filepath.Join(BinDir, target.Label.Subrepo, target.Label.PackageName)
	}
	return filepath.Join(GenDir, target.Label.Subrepo, target.Label.PackageName)
}

// ExecDir returns the exec directory for this target, e.g.
// //mickey/donald:goofy -> plz-out/exec/mickey/donald/goofy
func (target *BuildTarget) ExecDir() string {
	return filepath.Join(ExecDir, target.Label.Subrepo, target.Label.PackageName, target.Label.Name)
}

// TestDir returns the test directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy._test/run_1
// This is different to TmpDir so we run tests in a clean environment
// and to facilitate containerising tests.
func (target *BuildTarget) TestDir(runNumber int) string {
	return filepath.Join(target.TestDirs(), fmt.Sprint("run_", runNumber))
}

// TestLockFile returns the lock filename for the target's test stage.
func (target *BuildTarget) TestLockFile(runNumber int) string {
	return target.TestDir(runNumber) + lockFileSuffix
}

// TestDirs contains the parent directory of all the test run directories above
func (target *BuildTarget) TestDirs() string {
	return filepath.Join(TmpDir, target.Label.Subrepo, target.Label.PackageName, target.Label.Name+testDirSuffix)
}

// IsTest returns whether or not the target is a test target i.e. has its Test field populated
func (target *BuildTarget) IsTest() bool {
	return target.Test != nil
}

// CompleteRun completes a run and returns true if this was the last run
func (target *BuildTarget) CompleteRun(state *BuildState) bool {
	target.mutex.Lock()
	defer target.mutex.Unlock()

	target.completedRuns++
	return target.completedRuns == state.NumTestRuns
}

// TestResultsFile returns the output results file for tests for this target.
func (target *BuildTarget) TestResultsFile() string {
	return filepath.Join(target.OutDir(), ".test_results_"+target.Label.Name)
}

// CoverageFile returns the output coverage file for tests for this target.
func (target *BuildTarget) CoverageFile() string {
	return filepath.Join(target.OutDir(), ".test_coverage_"+target.Label.Name)
}

// AddTestResults adds results to the target
func (target *BuildTarget) AddTestResults(results TestSuite) {
	target.mutex.Lock()
	defer target.mutex.Unlock()

	if len(target.Test.Results.TestCases) == 0 {
		target.Test.Results.Cached = results.Cached // On the first run we take whatever this is
	} else {
		target.Test.Results.Cached = target.Test.Results.Cached && results.Cached
	}
	target.Test.Results.Collapse(results)
}

// StartTestSuite sets the initial properties on the result test suite
func (target *BuildTarget) StartTestSuite() {
	target.mutex.Lock()
	defer target.mutex.Unlock()

	// If the results haven't been set yet, set them
	if target.Test.Results == nil {
		target.Test.Results = &TestSuite{
			Package:   strings.ReplaceAll(target.Label.PackageName, "/", "."),
			Name:      target.Label.Name,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
}

// AllSourcePaths returns all the source paths for this target
func (target *BuildTarget) AllSourcePaths(graph *BuildGraph) []string {
	return target.allSourcePaths(graph, BuildInput.Paths)
}

// AllSourceFullPaths returns all the source paths for this target, with a leading
// plz-out/gen etc if appropriate.
func (target *BuildTarget) AllSourceFullPaths(graph *BuildGraph) []string {
	return target.allSourcePaths(graph, BuildInput.FullPaths)
}

// AllSourceLocalPaths returns the local part of all the source paths for this target,
// i.e. without this target's package in it.
func (target *BuildTarget) AllSourceLocalPaths(graph *BuildGraph) []string {
	return target.allSourcePaths(graph, BuildInput.LocalPaths)
}

type buildPathsFunc func(BuildInput, *BuildGraph) []string

func (target *BuildTarget) allSourcePaths(graph *BuildGraph, full buildPathsFunc) []string {
	ret := make([]string, 0, len(target.Sources))
	for _, source := range target.AllSources() {
		ret = append(ret, target.sourcePaths(graph, source, full)...)
	}
	return ret
}

// AllURLs returns all the URLs for this target.
// This should only be called if the target is a remote file.
// The URLs will have any embedded environment variables expanded according to the given config.
func (target *BuildTarget) AllURLs(state *BuildState) []string {
	env := GeneralBuildEnvironment(state)
	ret := make([]string, len(target.Sources))
	for i, s := range target.Sources {
		ret[i] = os.Expand(string(s.(URLLabel)), env.ReplaceEnvironment)
	}
	return ret
}

// resolveDependencies matches up all declared dependencies to the actual build targets.
// TODO(peterebden,tatskaari): Work out if we really want to have this and how the suite of *Dependencies functions
//
//	below should behave (preferably nicely).
func (target *BuildTarget) resolveDependencies(graph *BuildGraph, callback func(*BuildTarget) error) error {
	var g errgroup.Group
	target.mutex.RLock()
	for i := range target.dependencies {
		dep := &target.dependencies[i] // avoid using a loop variable here as it mutates each iteration
		if len(dep.deps) > 0 {
			continue // already done
		}
		g.Go(func() error {
			if err := target.resolveOneDependency(graph, dep); err != nil {
				return err
			}
			for _, d := range dep.deps {
				if err := callback(d); err != nil {
					return err
				}
			}
			return nil
		})
	}
	target.mutex.RUnlock()
	return g.Wait()
}

func (target *BuildTarget) resolveOneDependency(graph *BuildGraph, dep *depInfo) error {
	t := graph.WaitForTarget(*dep.declared)
	if t == nil {
		return fmt.Errorf("Couldn't find dependency %s", dep.declared)
	}
	dep.declared = &t.Label // saves memory by not storing the label twice once resolved

	labels, ok := t.provideFor(target)
	if !ok {
		target.mutex.Lock()
		defer target.mutex.Unlock()

		// Small optimisation to avoid re-looking-up the same target again.
		dep.deps = []*BuildTarget{t}
		return nil
	}

	deps := make([]*BuildTarget, 0, len(labels))
	for _, l := range labels {
		t := graph.WaitForTarget(l)
		if t == nil {
			return fmt.Errorf("%s depends on %s (provided by %s), however that target doesn't exist", target, l, t)
		}
		deps = append(deps, t)
	}

	target.mutex.Lock()
	defer target.mutex.Unlock()

	dep.deps = deps

	return nil
}

// MustResolveDependencies is exposed only for testing purposes.
// TODO(peterebden, tatskaari): See if we can get rid of this.
func (target *BuildTarget) ResolveDependencies(graph *BuildGraph) error {
	return target.resolveDependencies(graph, func(*BuildTarget) error { return nil })
}

// DeclaredDependencies returns all the targets this target declared any kind of dependency on (including sources and tools).
func (target *BuildTarget) DeclaredDependencies() []BuildLabel {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	ret := make(BuildLabels, len(target.dependencies))
	for i, dep := range target.dependencies {
		ret[i] = *dep.declared
	}
	sort.Sort(ret)
	return ret
}

// DeclaredDependenciesStrict returns the original declaration of this target's dependencies.
func (target *BuildTarget) DeclaredDependenciesStrict() []BuildLabel {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	ret := make(BuildLabels, 0, len(target.dependencies))
	for _, dep := range target.dependencies {
		if !dep.exported && !dep.source && !target.IsTool(*dep.declared) {
			ret = append(ret, *dep.declared)
		}
	}
	sort.Sort(ret)
	return ret
}

// Dependencies returns the resolved dependencies of this target.
func (target *BuildTarget) Dependencies() []*BuildTarget {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	ret := make(BuildTargets, 0, len(target.dependencies))
	for _, deps := range target.dependencies {
		for _, dep := range deps.deps {
			ret = append(ret, dep)
		}
	}
	sort.Sort(ret)
	return ret
}

// ExternalDependencies returns the non-internal dependencies of this target (i.e. not "_target#tag" ones).
func (target *BuildTarget) ExternalDependencies() []*BuildTarget {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	ret := make(BuildTargets, 0, len(target.dependencies))
	for _, deps := range target.dependencies {
		for _, dep := range deps.deps {
			if dep.Label.Parent() != target.Label {
				ret = append(ret, dep)
			} else {
				ret = append(ret, dep.ExternalDependencies()...)
			}
		}
	}
	sort.Sort(ret)
	return ret
}

// BuildDependencies returns the build-time dependencies of this target (i.e. not data, internal nor source).
func (target *BuildTarget) BuildDependencies() []*BuildTarget {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	ret := make(BuildTargets, 0, len(target.dependencies))
	for _, deps := range target.dependencies {
		if !deps.data && !deps.internal && !deps.source {
			for _, dep := range deps.deps {
				ret = append(ret, dep)
			}
		}
	}
	sort.Sort(ret)
	return ret
}

// ExportedDependencies returns any exported dependencies of this target.
func (target *BuildTarget) ExportedDependencies() []BuildLabel {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	ret := make(BuildLabels, 0, len(target.dependencies))
	for _, info := range target.dependencies {
		if info.exported {
			ret = append(ret, *info.declared)
		}
	}
	return ret
}

// DependenciesFor returns the dependencies that relate to a given label.
func (target *BuildTarget) DependenciesFor(label BuildLabel) []*BuildTarget {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	return target.dependenciesFor(label)
}

func (target *BuildTarget) dependenciesFor(label BuildLabel) []*BuildTarget {
	if info := target.dependencyInfo(label); info != nil {
		return info.deps
	} else if target.Label.Subrepo != "" && label.Subrepo == "" {
		// Can implicitly use the target's subrepo.
		label.Subrepo = target.Label.Subrepo
		return target.dependenciesFor(label)
	}
	return nil
}

// FinishBuild marks this target as having built.
func (target *BuildTarget) FinishBuild() {
	close(target.finishedBuilding)
}

// WaitForBuild blocks until this target has finished building.
func (target *BuildTarget) WaitForBuild() {
	<-target.finishedBuilding
}

// DeclaredOutputs returns the outputs from this target's original declaration.
// Hence it's similar to Outputs() but without the resolving of other rule names.
func (target *BuildTarget) DeclaredOutputs() []string {
	return target.outputs
}

// DeclaredNamedOutputs returns the named outputs from this target's original declaration.
func (target *BuildTarget) DeclaredNamedOutputs() map[string][]string {
	return target.namedOutputs
}

// DeclaredOutputNames is a convenience function to return the names of the declared
// outputs in a consistent order.
func (target *BuildTarget) DeclaredOutputNames() []string {
	ret := make([]string, 0, len(target.namedOutputs))
	for name := range target.namedOutputs {
		ret = append(ret, name)
	}
	sort.Strings(ret)
	return ret
}

// DeclaredNamedSources returns the named sources from this target's original declaration.
func (target *BuildTarget) DeclaredNamedSources() map[string][]string {
	ret := make(map[string][]string, len(target.NamedSources))
	for k, v := range target.NamedSources {
		ret[k] = make([]string, len(v))
		for i, bi := range v {
			ret[k][i] = bi.String()
		}
	}
	return ret
}

// DeclaredSourceNames is a convenience function to return the names of the declared
// sources in a consistent order.
func (target *BuildTarget) DeclaredSourceNames() []string {
	ret := make([]string, 0, len(target.NamedSources))
	for name := range target.NamedSources {
		ret = append(ret, name)
	}
	sort.Strings(ret)
	return ret
}

func (target *BuildTarget) filegroupOutputs(srcs []BuildInput) []string {
	ret := make([]string, 0, len(srcs))
	// Filegroups just re-output their inputs.
	for _, src := range srcs {
		if namedLabel, ok := src.(AnnotatedOutputLabel); ok {
			// Bit of a hack, but this needs different treatment from either of the others.
			for _, dep := range target.DependenciesFor(namedLabel.BuildLabel) {
				ret = append(ret, dep.NamedOutputs(namedLabel.Annotation)...)
			}
		} else if label, ok := src.nonOutputLabel(); !ok {
			ret = append(ret, src.LocalPaths(nil)[0])
		} else {
			for _, dep := range target.DependenciesFor(label) {
				ret = append(ret, dep.Outputs()...)
			}
		}
	}
	return ret
}

// Outputs returns a slice of all the outputs of this rule.
func (target *BuildTarget) Outputs() []string {
	var ret []string
	if target.IsFilegroup {
		ret = target.filegroupOutputs(target.AllSources())
	} else {
		// Must really copy the slice before sorting it ([:] is too shallow)
		ret = make([]string, len(target.outputs))
		copy(ret, target.outputs)
	}
	if target.namedOutputs != nil {
		for _, outputs := range target.namedOutputs {
			ret = append(ret, outputs...)
		}
	}
	sort.Strings(ret)
	return ret
}

// FullOutputs returns a slice of all the outputs of this rule with the target's output directory prepended.
func (target *BuildTarget) FullOutputs() []string {
	outs := target.Outputs()
	outDir := target.OutDir()
	for i, out := range outs {
		outs[i] = filepath.Join(outDir, out)
	}
	return outs
}

// AllOutputs returns a slice of all the outputs of this rule, including any output directories.
// Outs are passed through GetTmpOutput as appropriate.
func (target *BuildTarget) AllOutputs() []string {
	outs := target.Outputs()
	for i, out := range outs {
		outs[i] = target.GetTmpOutput(out)
	}
	for _, out := range target.OutputDirectories {
		outs = append(outs, out.Dir())
	}
	return outs
}

// NamedOutputs returns a slice of all the outputs of this rule with a given name.
// If the name is not declared by this rule it panics.
func (target *BuildTarget) NamedOutputs(name string) []string {
	if target.IsFilegroup {
		if target.NamedSources == nil {
			return nil
		}
		if srcs, present := target.NamedSources[name]; present {
			return target.filegroupOutputs(srcs)
		}
		return nil
	}
	if target.namedOutputs == nil {
		return nil
	}
	if outs, present := target.namedOutputs[name]; present {
		return outs
	}
	return nil
}

// GetTmpOutput takes the original output filename as an argument, and returns a temporary output
// filename(plz-out/tmp/) if output has the same name as the package, this avoids the name conflict issue
func (target *BuildTarget) GetTmpOutput(parseOutput string) string {
	if target.IsFilegroup {
		return parseOutput // Filegroups never need this.
	} else if parseOutput == target.Label.PackageName {
		return parseOutput + tempOutputSuffix
	} else if target.Label.PackageName == "" && target.HasSource(parseOutput) {
		// This also fixes the case where source and output are the same, which can happen
		// when we're in the root directory.
		return parseOutput + tempOutputSuffix
	}
	return parseOutput
}

// GetTmpOutputAll returns a slice of all the temporary outputs this is used in setting up environment for outputs,
// e.g: OUTS, OUT
func (target *BuildTarget) GetTmpOutputAll(parseOutputs []string) []string {
	tmpOutputs := make([]string, len(parseOutputs))
	for i, out := range parseOutputs {
		tmpOutputs[i] = target.GetTmpOutput(out)
	}
	return tmpOutputs
}

// GetRealOutput returns the real output name for a filename that might have been a temporary output
// (i.e as returned by GetTmpOutput).
func (target *BuildTarget) GetRealOutput(output string) string {
	if strings.HasSuffix(output, tempOutputSuffix) {
		real := strings.TrimSuffix(output, tempOutputSuffix)
		// Check this isn't a file that just happens to be named the same way
		if target.GetTmpOutput(real) == output {
			return real
		}
	}
	return output
}

// SourcePaths returns the source paths for a given set of sources.
func (target *BuildTarget) SourcePaths(graph *BuildGraph, sources []BuildInput) []string {
	ret := make([]string, 0, len(sources))
	for _, source := range sources {
		ret = append(ret, target.sourcePaths(graph, source, BuildInput.Paths)...)
	}
	return ret
}

// sourcePaths returns the source paths for a single source.
func (target *BuildTarget) sourcePaths(graph *BuildGraph, source BuildInput, f buildPathsFunc) []string {
	if label, ok := source.nonOutputLabel(); ok {
		ret := []string{}
		for _, providedLabel := range graph.TargetOrDie(label).ProvideFor(target) {
			ret = append(ret, f(providedLabel, graph)...)
		}
		return ret
	}
	return f(source, graph)
}

// CanSee returns true if target can see the given dependency, or false if not.
func (target *BuildTarget) CanSee(state *BuildState, dep *BuildTarget) bool {
	return target.Label.CanSee(state, dep)
}

// CheckDependencyVisibility checks that all declared dependencies of this target are visible to it.
// Returns an error if not, or nil if all's well.
func (target *BuildTarget) CheckDependencyVisibility(state *BuildState) error {
	for _, d := range target.dependencies {
		dep := state.Graph.TargetOrDie(*d.declared)
		if !target.CanSee(state, dep) {
			return fmt.Errorf("Target %s isn't visible to %s", dep.Label, target.Label)
		} else if dep.TestOnly && !(target.IsTest() || target.TestOnly) {
			if target.Label.isExperimental(state) {
				log.Info("Test-only restrictions suppressed for %s since %s is in the experimental tree", dep.Label, target.Label)
			} else {
				return fmt.Errorf("Target %s can't depend on %s, it's marked test_only", target.Label, dep.Label)
			}
		}
	}
	return nil
}

// CheckDuplicateOutputs checks if any of the outputs of this target duplicate one another.
// Returns an error if so, or nil if all's well.
func (target *BuildTarget) CheckDuplicateOutputs() error {
	outputs := map[string]struct{}{}
	for _, output := range target.Outputs() {
		if _, present := outputs[output]; present {
			return fmt.Errorf("Target %s declares output file %s multiple times", target.Label, output)
		}
		outputs[output] = struct{}{}
	}
	return nil
}

// CheckTargetOwnsBuildOutputs checks that any outputs to this rule output into directories this of this package.
func (target *BuildTarget) CheckTargetOwnsBuildOutputs(state *BuildState) error {
	// Skip this check for sub-repos because sub-repos are currently outputted into plz-gen so the output might also
	// be a sub-repo that contains a package. This isn't the best solution but we can't fix this without reworking
	// how sub-repos are done.
	if target.Subrepo != nil {
		return nil
	}

	for _, output := range target.Outputs() {
		targetPackage := target.Label.PackageName
		out := filepath.Join(targetPackage, output)

		if fs.IsPackage(state.Config.Parse.BuildFileName, out) {
			return fmt.Errorf("trying to output file %s, but that directory is another package", out)
		}

		// If the output is just a file in the package root, we don't need to check anything else.
		if filepath.Dir(output) == "." {
			continue
		}

		pkg := FindOwningPackage(state, out)
		if targetPackage != pkg.PackageName {
			return fmt.Errorf("trying to output file %s, but that directory belongs to another package (%s)", out, pkg.PackageName)
		}
	}
	return nil
}

// CheckTargetOwnsBuildInputs checks that any file inputs to this rule belong to this package.
func (target *BuildTarget) CheckTargetOwnsBuildInputs(state *BuildState) error {
	for _, input := range target.AllSources() {
		if err := target.checkTargetOwnsBuildInput(state, input); err != nil {
			return err
		}
	}

	for _, input := range target.AllData() {
		if err := target.checkTargetOwnsBuildInput(state, input); err != nil {
			return err
		}
	}
	return nil
}

func (target *BuildTarget) checkTargetOwnsBuildInput(state *BuildState, input BuildInput) error {
	if input, ok := input.(FileLabel); ok {
		for _, f := range input.Paths(state.Graph) {
			if err := target.checkTargetOwnsFileAndSubDirectories(state, f); err != nil {
				return err
			}
		}
	}
	return nil
}

func (target *BuildTarget) checkTargetOwnsFileAndSubDirectories(state *BuildState, file string) error {
	pkg := FindOwningPackage(state, file)
	if target.Label.PackageName != pkg.PackageName {
		return fmt.Errorf("package %s is trying to use file %s, but that belongs to another package (%s)", target.Label.PackageName, file, pkg.PackageName)
	}

	if fs.IsDirectory(file) {
		err := fs.Walk(file, func(name string, isDir bool) error {
			if isDir && fs.IsPackage(state.Config.Parse.BuildFileName, name) {
				return fmt.Errorf("cannot include %s as it contains subpackage %s", file, name)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckSecrets checks that this target's secrets are available.
// We run this check before building because we don't attempt to copy them, but any rule
// requiring them will presumably fail if they aren't available.
// Returns an error if any aren't.
func (target *BuildTarget) CheckSecrets() error {
	for _, secret := range target.AllSecrets() {
		if path := fs.ExpandHomePath(secret); !PathExists(path) {
			return fmt.Errorf("Path %s doesn't exist; it's required to build %s", secret, target.Label)
		}
	}
	return nil
}

// AllSecrets returns all the sources of this rule.
func (target *BuildTarget) AllSecrets() []string {
	ret := target.Secrets[:]
	if target.NamedSecrets != nil {
		keys := make([]string, 0, len(target.NamedSecrets))
		for k := range target.NamedSecrets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ret = append(ret, target.NamedSecrets[k]...)
		}
	}
	return ret
}

// HasDependency checks if a target already depends on this label.
func (target *BuildTarget) HasDependency(label BuildLabel) bool {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	return target.dependencyInfo(label) != nil
}

// resolveDependency resolves a particular dependency on a target.
// TODO(jpoole): this is only used by tests: remove
func (target *BuildTarget) resolveDependency(label BuildLabel, dep *BuildTarget) {
	target.mutex.Lock()
	defer target.mutex.Unlock()
	info := target.dependencyInfo(label)
	if info == nil {
		target.dependencies = append(target.dependencies, depInfo{declared: &label})
		info = &target.dependencies[len(target.dependencies)-1]
	}
	if dep != nil {
		info.deps = append(info.deps, dep)
	}
	info.resolved = true
}

// dependencyInfo returns the information about a declared dependency, or nil if the target doesn't have it.
func (target *BuildTarget) dependencyInfo(label BuildLabel) *depInfo {
	for i, info := range target.dependencies {
		if *info.declared == label {
			return &target.dependencies[i]
		}
	}
	return nil
}

// IsSourceOnlyDep returns true if the given dependency was only declared on the srcs of the target.
func (target *BuildTarget) IsSourceOnlyDep(label BuildLabel) bool {
	info := target.dependencyInfo(label)
	return info != nil && info.source
}

// State returns the target's current state.
func (target *BuildTarget) State() BuildTargetState {
	return BuildTargetState(atomic.LoadInt32(&target.state))
}

// SetState sets a target's current state.
func (target *BuildTarget) SetState(state BuildTargetState) {
	atomic.StoreInt32(&target.state, int32(state))
}

// SyncUpdateState oves the target's state from before to after via a lock.
// Returns true if successful, false if not (which implies something else changed the state first).
// The nature of our build graph ensures that most transitions are only attempted by
// one thread simultaneously, but this one can be attempted by several at once
// (eg. if a depends on b and c, which finish building simultaneously, they race to queue a).
func (target *BuildTarget) SyncUpdateState(before, after BuildTargetState) bool {
	return atomic.CompareAndSwapInt32(&target.state, int32(before), int32(after))
}

// AddLabel adds the given label to this target if it doesn't already have it.
func (target *BuildTarget) AddLabel(label string) {
	if !target.HasLabel(label) {
		target.Labels = append(target.Labels, label)
	}
}

// HasLabel returns true if target has the given label.
func (target *BuildTarget) HasLabel(label string) bool {
	for _, l := range target.Labels {
		if match(label, l) {
			return true
		}
	}
	return label == "test" && target.IsTest()
}

// match returns true if the given label matches the given pattern.
func match(pattern, s string) bool {
	if pattern == s {
		return true
	}
	if strings.HasSuffix(pattern, "*") && strings.HasPrefix(s, pattern[:len(pattern)-1]) {
		return true
	}
	return false
}

// PrefixedLabels returns all labels of this target with the given prefix.
func (target *BuildTarget) PrefixedLabels(prefix string) []string {
	ret := []string{}
	for _, l := range target.Labels {
		if strings.HasPrefix(l, prefix) {
			ret = append(ret, strings.TrimPrefix(l, prefix))
		}
	}
	return ret
}

// HasAnyLabel returns true if target has any of these labels.
func (target *BuildTarget) HasAnyLabel(labels []string) bool {
	for _, label := range labels {
		if target.HasLabel(label) {
			return true
		}
	}
	return false
}

// HasAllLabels returns true if target has all of these labels.
func (target *BuildTarget) HasAllLabels(labels []string) bool {
	for _, label := range labels {
		if !target.HasLabel(label) {
			return false
		}
	}
	return true
}

// ShouldInclude handles the typical include/exclude logic for a target's labels; returns true if
// target has any include label and not an exclude one.
// Each include/exclude can have multiple comma-separated labels; in this case, all of the labels
// in a given group must match.
func (target *BuildTarget) ShouldInclude(includes, excludes []string) bool {
	if len(includes) == 0 && len(excludes) == 0 {
		return true
	}

	// Include by default if no includes are specified.
	shouldInclude := len(includes) == 0
	for _, include := range includes {
		if target.HasAllLabels(strings.Split(include, ",")) {
			shouldInclude = true
			break
		}
	}
	for _, exclude := range excludes {
		if target.HasAllLabels(strings.Split(exclude, ",")) {
			shouldInclude = false
			break
		}
	}
	return shouldInclude
}

// AddProvide adds a new provide entry to this target.
func (target *BuildTarget) AddProvide(language string, labels []BuildLabel) {
	if target.Provides == nil {
		target.Provides = map[string][]BuildLabel{language: labels}
	} else {
		target.Provides[language] = labels
	}
}

// ProvideFor returns the build label that we'd provide for the given target.
func (target *BuildTarget) ProvideFor(other *BuildTarget) []BuildLabel {
	if p, ok := target.provideFor(other); ok {
		return p
	}
	return []BuildLabel{target.Label}
}

// provideFor is like ProvideFor but returns an empty slice if there is a direct dependency.
// It's a small optimisation to save allocating extra slices.
func (target *BuildTarget) provideFor(other *BuildTarget) ([]BuildLabel, bool) {
	target.mutex.RLock()
	defer target.mutex.RUnlock()
	if target.Provides == nil || len(other.Requires) == 0 {
		return nil, false
	}
	// Never do this if the other target has a data or tool dependency on us.
	for _, data := range other.Data {
		if label, ok := data.Label(); ok && label == target.Label {
			return nil, false
		}
	}
	if other.IsTool(target.Label) {
		return nil, false
	}
	var ret []BuildLabel
	found := false
	for _, require := range other.Requires {
		if label, present := target.Provides[require]; present {
			if ret == nil {
				ret = make([]BuildLabel, 0, len(other.Requires))
			}
			ret = append(ret, label...)
			found = true
		}
	}
	return ret, found
}

// UnprefixedHashes returns the hashes for the target without any prefixes;
// they are allowed to have optional prefixes before a colon which aren't taken
// into account for the resulting hash.
func (target *BuildTarget) UnprefixedHashes() []string {
	hashes := target.Hashes[:]
	for i, h := range hashes {
		if index := strings.LastIndexByte(h, ':'); index != -1 {
			hashes[i] = strings.TrimSpace(h[index+1:])
		}
	}
	return hashes
}

// HashLastModified is whether we should hash the last modified times for this target
func (target *BuildTarget) HashLastModified() bool {
	return target.IsFilegroup && target.HasLabel("fg:hash-modified-time")
}

// AddSource adds a source to the build target, deduplicating against existing entries.
func (target *BuildTarget) AddSource(source BuildInput) {
	target.Sources = target.addSource(target.Sources, source)
}

func (target *BuildTarget) addSource(sources []BuildInput, source BuildInput) []BuildInput {
	for _, src := range sources {
		if source == src {
			return sources
		}
	}
	// Add a dependency if this is not just a file.
	if label, ok := source.Label(); ok {
		target.AddMaybeExportedDependency(label, false, true, false)
	}
	return append(sources, source)
}

// AddSecret adds a secret to the build target, deduplicating against existing entries.
func (target *BuildTarget) AddSecret(secret string) {
	target.Secrets = target.addSecret(target.Secrets, secret)
}

func (target *BuildTarget) addSecret(secrets []string, secret string) []string {
	for _, existing := range secrets {
		if existing == secret {
			return secrets
		}
	}
	return append(secrets, secret)
}

// AddNamedSource adds a source to the target which is tagged with a particular name.
// For example, C++ rules add sources tagged as "sources" and "headers" to distinguish
// two conceptually different kinds of input.
func (target *BuildTarget) AddNamedSource(name string, source BuildInput) {
	if target.NamedSources == nil {
		target.NamedSources = make(map[string][]BuildInput)
	}
	target.NamedSources[name] = target.addSource(target.NamedSources[name], source)
}

// AddNamedSecret adds a secret to the target which is tagged with a particular name.
// These will be made available in the environment at runtime, with key-format "SECRETS_<NAME>".
func (target *BuildTarget) AddNamedSecret(name string, secret string) {
	if target.NamedSecrets == nil {
		target.NamedSecrets = make(map[string][]string)
	}
	target.NamedSecrets[name] = target.addSecret(target.NamedSecrets[name], secret)
}

// AddTool adds a new tool to the target.
func (target *BuildTarget) AddTool(tool BuildInput) {
	target.Tools = append(target.Tools, tool)
	if label, ok := tool.Label(); ok {
		target.AddDependency(label)
	}
}

// AddTestTool adds a new test tool to the target.
func (target *BuildTarget) AddTestTool(tool BuildInput) {
	target.Test.tools = append(target.Test.tools, tool)
	if label, ok := tool.Label(); ok {
		target.AddDependency(label)
	}
}

// AddDebugTool adds a new tool for debugging the target.
func (target *BuildTarget) AddDebugTool(tool BuildInput) {
	if target.Debug == nil {
		target.Debug = new(DebugFields)
	}
	target.Debug.tools = append(target.Debug.tools, tool)
	if label, ok := tool.Label(); ok {
		target.AddDependency(label)
	}
}

// AllTestTools returns all the test tool paths for this rule.
func (target *BuildTarget) AllTestTools() []BuildInput {
	if target.Test.namedTools == nil {
		return target.Test.tools
	}
	return target.allBuildInputs(target.Test.tools, target.Test.namedTools)
}

// NamedTestTools returns all named test tools
func (target *BuildTarget) NamedTestTools() map[string][]BuildInput {
	return target.Test.namedTools
}

// AllDebugTools returns all the debug tool paths for this rule.
func (target *BuildTarget) AllDebugTools() []BuildInput {
	if target.Debug == nil {
		return nil
	}
	if target.Debug.namedTools == nil {
		return target.Debug.tools
	}
	return target.allBuildInputs(target.Debug.tools, target.Debug.namedTools)
}

// AddDatum adds a new item of data to the target.
func (target *BuildTarget) AddDatum(datum BuildInput) {
	target.Data = append(target.Data, datum)
	if label, ok := datum.Label(); ok {
		target.AddDependency(label)
		target.dependencyInfo(label).data = true
	}
}

// AddNamedDatum adds a data file to the target which is tagged with a particular name.
func (target *BuildTarget) AddNamedDatum(name string, datum BuildInput) {
	if target.NamedData == nil {
		target.NamedData = make(map[string][]BuildInput)
	}
	target.NamedData[name] = append(target.NamedData[name], datum)
	if label, ok := datum.Label(); ok {
		target.AddDependency(label)
		target.dependencyInfo(label).data = true
	}
}

// AddDebugDatum adds a new item of debug data to the target.
func (target *BuildTarget) AddDebugDatum(datum BuildInput) {
	if target.Debug == nil {
		target.Debug = new(DebugFields)
	}
	target.Debug.data = append(target.Debug.data, datum)
	if label, ok := datum.Label(); ok {
		target.AddDependency(label)
		target.dependencyInfo(label).data = true
	}
}

// AddDebugNamedDatum adds a new item of debug data to the target which is tagged with a particular name.
func (target *BuildTarget) AddDebugNamedDatum(name string, datum BuildInput) {
	if target.Debug == nil {
		target.Debug = new(DebugFields)
	}
	if target.Debug.namedData == nil {
		target.Debug.namedData = make(map[string][]BuildInput)
	}
	target.Debug.namedData[name] = append(target.Debug.namedData[name], datum)
	if label, ok := datum.Label(); ok {
		target.AddDependency(label)
		target.dependencyInfo(label).data = true
	}
}

// AddNamedTool adds a new tool to the target.
func (target *BuildTarget) AddNamedTool(name string, tool BuildInput) {
	if target.namedTools == nil {
		target.namedTools = make(map[string][]BuildInput)
	}
	target.namedTools[name] = append(target.namedTools[name], tool)
	if label, ok := tool.Label(); ok {
		target.AddDependency(label)
	}
}

// AddNamedTestTool adds a new test tool to the target.
func (target *BuildTarget) AddNamedTestTool(name string, tool BuildInput) {
	if target.Test == nil {
		target.Test = new(TestFields)
	}
	if target.Test.namedTools == nil {
		target.Test.namedTools = make(map[string][]BuildInput)
	}
	target.Test.namedTools[name] = append(target.Test.namedTools[name], tool)
	if label, ok := tool.Label(); ok {
		target.AddDependency(label)
	}
}

// AddNamedDebugTool adds a new debug tool to the target.
func (target *BuildTarget) AddNamedDebugTool(name string, tool BuildInput) {
	if target.Debug == nil {
		target.Debug = new(DebugFields)
	}
	if target.Debug.namedTools == nil {
		target.Debug.namedTools = make(map[string][]BuildInput)
	}
	target.Debug.namedTools[name] = append(target.Debug.namedTools[name], tool)
	if label, ok := tool.Label(); ok {
		target.AddDependency(label)
	}
}

// AddCommand adds a new config-specific command to this build target.
// Adding a general command is still done by simply setting the Command member.
func (target *BuildTarget) AddCommand(config, command string) {
	if target.Command != "" {
		panic(fmt.Sprintf("Adding named command %s to %s, but it already has a general command set", config, target.Label))
	} else if target.Commands == nil {
		target.Commands = map[string]string{config: command}
	} else {
		target.Commands[config] = command
	}
}

// AddTestCommand adds a new config-specific test command to this build target.
// Adding a general command is still done by simply setting the Command member.
func (target *BuildTarget) AddTestCommand(config, command string) {
	if target.Test.Command != "" {
		panic(fmt.Sprintf("Adding named test command %s to %s, but it already has a general test command set", config, target.Label))
	} else if target.Test.Commands == nil {
		target.Test.Commands = map[string]string{config: command}
	} else {
		target.Test.Commands[config] = command
	}
}

// GetCommand returns the command we should use to build this target for the current config.
func (target *BuildTarget) GetCommand(state *BuildState) string {
	return target.getCommand(state, target.Commands, target.Command)
}

// GetCommandConfig returns the command we should use to build this target for the given config.
func (target *BuildTarget) GetCommandConfig(config string) string {
	if config == "" {
		return target.Command
	}
	return target.Commands[config]
}

// GetTestCommand returns the command we should use to test this target for the current config.
func (target *BuildTarget) GetTestCommand(state *BuildState) string {
	return target.getCommand(state, target.Test.Commands, target.Test.Command)
}

func (target *BuildTarget) getCommand(state *BuildState, commands map[string]string, singleCommand string) string {
	if commands == nil {
		return singleCommand
	} else if command, present := commands[state.Config.Build.Config]; present {
		return command // Has command for current config, good
	} else if command, present := commands[state.Config.Build.FallbackConfig]; present {
		return command // Has command for default config, fall back to that
	}
	// Oh dear, target doesn't have any matching config. Panicking is a bit heavy here, instead
	// fall back to an arbitrary (but consistent) one.
	highestCommand := ""
	highestConfig := ""
	for config, command := range commands {
		if config > highestConfig {
			highestConfig = config
			highestCommand = command
		}
	}
	log.Warning("%s doesn't have a command for %s (or %s), falling back to %s",
		target.Label, state.Config.Build.Config, state.Config.Build.FallbackConfig, highestConfig)
	return highestCommand
}

// AllSources returns all the sources of this rule.
func (target *BuildTarget) AllSources() []BuildInput {
	if target.NamedSources == nil {
		return target.Sources
	}
	return target.allBuildInputs(target.Sources, target.NamedSources)
}

func (target *BuildTarget) allBuildInputs(unnamed []BuildInput, named map[string][]BuildInput) []BuildInput {
	ret := unnamed
	keys := make([]string, 0, len(named))
	for k := range named {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ret = append(ret, named[k]...)
	}

	return ret
}

// AllLocalSourcePaths returns all the "local" sources of this rule, i.e. all sources that are
// actually sources in the repo, not other rules or system srcs etc.
func (target *BuildTarget) AllLocalSourcePaths() []string {
	return target.allLocalSourcePaths(BuildInput.Paths)
}

func (target *BuildTarget) AllLocalSourceLocalPaths() []string {
	return target.allLocalSourcePaths(BuildInput.LocalPaths)
}

func (target *BuildTarget) allLocalSourcePaths(full buildPathsFunc) []string {
	srcs := target.AllSources()
	ret := make([]string, 0, len(srcs))
	for _, src := range srcs {
		if file, ok := src.(FileLabel); ok {
			ret = append(ret, full(file, nil)[0])
		}
	}
	return ret
}

// HasSource returns true if this target has the given file as a source (named or not, or data).
func (target *BuildTarget) HasSource(source string) bool {
	for _, src := range append(target.AllSources(), target.AllData()...) {
		// Check for both the source matching and a prefix match indicating it's a directory with the file within.
		if s := src.String(); s == source || strings.HasPrefix(source, s+"/") {
			return true
		}
	}
	return false
}

// HasAbsoluteSource returns true if this target has the given file as a source (or data).
// The input source includes the target's package name.
func (target *BuildTarget) HasAbsoluteSource(source string) bool {
	return target.HasSource(strings.TrimPrefix(source, target.Label.PackageName+"/"))
}

// AllData returns all the runtime data of this rule.
func (target *BuildTarget) AllData() []BuildInput {
	if target.NamedData == nil {
		return target.Data
	}

	return target.allBuildInputs(target.Data, target.NamedData)
}

// AllDebugData returns all the data for debugging this rule.
func (target *BuildTarget) AllDebugData() []BuildInput {
	if target.Debug == nil {
		return nil
	}
	if target.Debug.namedData == nil {
		return target.Debug.data
	}
	return target.allBuildInputs(target.Debug.data, target.Debug.namedData)
}

// DebugData returns unnamed data for debugging this rule.
func (target *BuildTarget) DebugData() []BuildInput {
	if target.Debug == nil {
		return nil
	}
	return target.Debug.data
}

// DebugNamedData returns named data for debugging this rule.
func (target *BuildTarget) DebugNamedData() map[string][]BuildInput {
	if target.Debug == nil {
		return nil
	}
	return target.Debug.namedData
}

// AllTools returns all the tools for this rule in some canonical order.
func (target *BuildTarget) AllTools() []BuildInput {
	if target.namedTools == nil {
		return target.Tools
	}
	return target.allBuildInputs(target.Tools, target.namedTools)
}

// ToolNames returns an ordered list of tool names.
func (target *BuildTarget) ToolNames() []string {
	ret := make([]string, 0, len(target.namedTools))
	for name := range target.namedTools {
		ret = append(ret, name)
	}
	sort.Strings(ret)
	return ret
}

// NamedTools returns the tools with the given name.
func (target *BuildTarget) NamedTools(name string) []BuildInput {
	return target.namedTools[name]
}

func (target *BuildTarget) AllNamedTools() map[string][]BuildInput {
	return target.namedTools
}

// AddDependency adds a dependency to this target. It deduplicates against any existing deps.
func (target *BuildTarget) AddDependency(dep BuildLabel) {
	target.AddMaybeExportedDependency(dep, false, false, false)
}

// HintDependencies allocates space for at least the given number of dependencies without reallocating.
func (target *BuildTarget) HintDependencies(n int) {
	target.dependencies = slices.Grow(target.dependencies, n)
}

// AddMaybeExportedDependency adds a dependency to this target which may be exported. It deduplicates against any existing deps.
func (target *BuildTarget) AddMaybeExportedDependency(dep BuildLabel, exported, source, internal bool) {
	if dep == target.Label {
		log.Fatalf("Attempted to add %s as a dependency of itself.\n", dep)
	}
	info := target.dependencyInfo(dep)
	if info == nil {
		target.dependencies = append(target.dependencies, depInfo{declared: &dep, exported: exported, source: source, internal: internal})
	} else {
		info.exported = info.exported || exported
		info.source = info.source && source
		info.internal = info.internal && internal
		info.data = false // It's not *only* data any more.
	}
}

// IsTool returns true if the given build label is a tool used by this target.
func (target *BuildTarget) IsTool(tool BuildLabel) bool {
	for _, t := range target.Tools {
		if label, ok := t.Label(); ok && label == tool {
			return true
		}
	}
	for _, tools := range target.namedTools {
		for _, t := range tools {
			if label, ok := t.Label(); ok && label == tool {
				return true
			}
		}
	}
	return false
}

// toolPath returns a path to this target when used as a tool.
func (target *BuildTarget) toolPath(abs bool, namedOutput string) string {
	outToolPath := func(outputs ...string) string {
		ret := make([]string, len(outputs))
		for i, o := range outputs {
			if abs {
				ret[i] = filepath.Join(RepoRoot, target.OutDir(), o)
			} else {
				ret[i] = filepath.Join(target.PackageDir(), o)
			}
		}
		return strings.Join(ret, " ")
	}

	if namedOutput != "" {
		if o, ok := target.EntryPoints[namedOutput]; ok {
			return outToolPath(o)
		}
		if outs, ok := target.namedOutputs[namedOutput]; ok {
			return outToolPath(outs...)
		}
		panic(fmt.Sprintf("%v has no named output or entry point %v", target.Label, namedOutput))
	}
	return outToolPath(target.Outputs()...)
}

// AddOutput adds a new output to the target if it's not already there.
func (target *BuildTarget) AddOutput(output string) {
	target.outputs = target.insert(target.outputs, output)
}

// AddOptionalOutput adds a new optional output to the target if it's not already there.
func (target *BuildTarget) AddOptionalOutput(output string) {
	target.OptionalOutputs = target.insert(target.OptionalOutputs, output)
}

// AddTestOutput adds a new test output to the target if it's not already there.
func (target *BuildTarget) AddTestOutput(output string) {
	target.Test.Outputs = target.insert(target.Test.Outputs, output)
}

// AddNamedOutput adds a new output to the target under a named group.
// No attempt to deduplicate against unnamed outputs is currently made.
func (target *BuildTarget) AddNamedOutput(name, output string) {
	if target.namedOutputs == nil {
		target.namedOutputs = make(map[string][]string)
	}
	target.namedOutputs[name] = target.insert(target.namedOutputs[name], output)
}

// AddEntryPoint adds a new entry point to the target. It panics if the given entry
// point name is disallowed in the context of this build target, or if an entry point
// with the same name already exists.
func (target *BuildTarget) AddEntryPoint(name, output string) {
	if target.EntryPoints == nil {
		target.EntryPoints = make(map[string]string)
	}
	if target.NamedOutputs(name) != nil {
		panic(fmt.Sprintf("%v already has a named output named %v; entry points may not have the same name as a named output", target.Label, name))
	}
	if target.IsFilegroup && target.NamedSources[name] != nil {
		panic(fmt.Sprintf("%v already has a named source named %v; entry points may not have the same name as a named source on a filegroup", target.Label, name))
	}
	if _, exists := target.EntryPoints[name]; exists {
		panic(fmt.Sprintf("%v already has an entry point named %v", target.Label, name))
	}
	target.EntryPoints[name] = output
}

// insert adds a string into a slice if it's not already there. Sorted order is maintained.
func (target *BuildTarget) insert(sl []string, s string) []string {
	if s == "" {
		panic("Cannot add an empty string as an output of a target")
	}
	s = strings.TrimPrefix(s, "./")
	for i, x := range sl {
		if s == x {
			// Already present.
			return sl
		} else if x > s {
			// Insert in this location. Make an attempt to be efficient.
			sl = append(sl, "")
			copy(sl[i+1:], sl[i:])
			sl[i] = s
			return sl
		}
	}
	return append(sl, s)
}

// AddLicence adds a licence to the target if it's not already there.
func (target *BuildTarget) AddLicence(licence string) {
	licence = strings.TrimSpace(licence)
	for _, l := range target.Licences {
		if l == licence {
			return
		}
	}
	target.Licences = append(target.Licences, licence)
}

// AddHash adds a new acceptable hash to the target.
func (target *BuildTarget) AddHash(hash string) {
	target.Hashes = append(target.Hashes, hash)
}

// AddRequire adds a new requirement to the target.
func (target *BuildTarget) AddRequire(require string) {
	target.Requires = append(target.Requires, require)
	// Requirements are also implicit labels
	target.AddLabel(require)
}

// OutMode returns the mode to set outputs of a target to.
func (target *BuildTarget) OutMode() os.FileMode {
	if target.IsBinary {
		return 0555
	}
	return 0444
}

// TargetBuildMetadataFileName returns the target build metadata file name for this target.
func (target *BuildTarget) TargetBuildMetadataFileName() string {
	return ".target_build_metadata_" + target.Label.Name
}

// StampFileName returns the stamp filename for this target.
func (target *BuildTarget) StampFileName() string {
	return ".stamp_" + target.Label.Name
}

// NeedCoverage returns true if this target should output coverage during a test
// for a particular invocation.
func (target *BuildTarget) NeedCoverage(state *BuildState) bool {
	if target.Test == nil {
		return false
	}
	return state.NeedCoverage && !target.Test.NoOutput && !target.HasAnyLabel(state.Config.Test.DisableCoverage)
}

// Parent finds the parent of a build target, or nil if the target is parentless.
// Note that this is a fairly informal relationship; we identify it by labels with the convention of
// a leading _ and trailing hashtag on child rules, rather than storing pointers between them in the graph.
// The parent returned, if any, will be the ultimate ancestor of the target.
func (target *BuildTarget) Parent(graph *BuildGraph) *BuildTarget {
	parent := target.Label.Parent()
	if parent == target.Label {
		return nil
	}
	return graph.Target(parent)
}

// HasParent returns true if the target has a parent rule that's not itself.
func (target *BuildTarget) HasParent() bool {
	return target.Label.HasParent()
}

// ShowProgress enables this target to display progress as it runs.
func (target *BuildTarget) ShowProgress() {
	target.showProgress.Store(true)
}

// ShouldShowProgress returns true if the target should display progress.
func (target *BuildTarget) ShouldShowProgress() bool {
	return target.showProgress.Load()
}

// ProgressDescription returns a description of what the target is doing as it runs.
// This is provided as a function to satisfy the process package.
func (target *BuildTarget) ProgressDescription() string {
	if target.State() >= Built && target.IsTest() {
		return "testing"
	}
	return target.BuildingDescription
}

// ShouldExitOnError returns true if the subprocess should exit when an error occurs.
func (target *BuildTarget) ShouldExitOnError() bool {
	return target.ExitOnError
}

// SetProgress sets the current progress of this target.
func (target *BuildTarget) SetProgress(progress float32) {
	target.Progress.Store(progress)
}

// BuildCouldModifyTarget will return true when the action of building this target could change the target itself e.g.
// by adding new outputs
func (target *BuildTarget) BuildCouldModifyTarget() bool {
	return target.PostBuildFunction != nil || len(target.OutputDirectories) > 0
}

// AddOutputDirectory adds an output directory to the target
func (target *BuildTarget) AddOutputDirectory(dir string) {
	target.OutputDirectories = append(target.OutputDirectories, OutputDirectory(dir))
}

// GetFileContent returns the file content, expanding it if it needs to
func (target *BuildTarget) GetFileContent(state *BuildState) (string, error) {
	return ReplaceSequences(state, target, target.FileContent)
}

// HasLinks returns true if the outputs are meant to be linked somewhere (i.e. via symlinks).
// This check is useful in deciding whether this target should be downloaded during remote execution or not.
func (target *BuildTarget) HasLinks(state *BuildState) bool {
	for _, prefix := range []string{"link:", "hlink:", "dlink:", "dhlink:"} {
		if labels := target.PrefixedLabels(prefix); len(labels) > 0 {
			return true
		}
	}
	if state.Config.ShouldLinkGeneratedSources() && target.HasLabel("codegen") {
		return true
	}
	return false
}

func (target *BuildTarget) PackageDir() string {
	if target.Subrepo != nil {
		return filepath.Join(target.Subrepo.PackageRoot, target.Label.PackageDir())
	}
	return target.Label.PackageDir()
}

// CheckLicences checks the target's licences against the accepted/rejected list.
// It returns the licence that was accepted and an error if it did not match.
func (target *BuildTarget) CheckLicences(config *Configuration) (string, error) {
	if len(target.Licences) == 0 {
		return "", nil
	}
	for _, licence := range target.Licences {
		for _, reject := range config.Licences.Reject {
			if strings.EqualFold(reject, licence) {
				return "", fmt.Errorf("Target %s is licensed %s, which is explicitly rejected for this repository", target.Label, licence)
			}
		}
		for _, accept := range config.Licences.Accept {
			if strings.EqualFold(accept, licence) {
				return licence, nil // Note licences are assumed to be an 'or', ie. any one of them can be accepted.
			}
		}
	}
	if len(config.Licences.Accept) > 0 {
		return "", fmt.Errorf("None of the licences for %s are accepted in this repository: %s", target.Label, strings.Join(target.Licences, ", "))
	}
	return "", nil
}

// BuildTargets makes a slice of build targets sortable by their labels.
type BuildTargets []*BuildTarget

func (slice BuildTargets) Len() int {
	return len(slice)
}
func (slice BuildTargets) Less(i, j int) bool {
	return slice[i].Label.Less(slice[j].Label)
}
func (slice BuildTargets) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
