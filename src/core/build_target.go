package core

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// OutDir is the root output directory for everything.
const OutDir string = "plz-out"

// TmpDir is the root of the temporary directory for building targets & running tests.
const TmpDir string = "plz-out/tmp"

// GenDir is the output directory for non-binary targets.
const GenDir string = "plz-out/gen"

// BinDir is the output directory for binary targets.
const BinDir string = "plz-out/bin"

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

// TestResultsDirLabel is a known label that indicates that the test will output results
// into a directory rather than a file. Please can internally handle either but the remote
// execution API requires that we specify which is which.
const TestResultsDirLabel = "test_results_dir"

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
	Data []BuildInput
	// Output files of this rule. All are paths relative to this package.
	outputs []string `name:"outs"`
	// Named output subsets of this rule. All are paths relative to this package but can be
	// captured separately; for example something producing C code might separate its outputs
	// into sources and headers.
	namedOutputs map[string][]string `name:"outs"`
	// Optional output files of this rule. Same as outs but aren't required to be produced always.
	// Can be glob patterns.
	OptionalOutputs []string `name:"optional_outs"`
	// Optional labels applied to this rule. Used for including/excluding rules.
	Labels []string
	// Shell command to run.
	Command string `name:"cmd" hide:"filegroup"`
	// Per-configuration shell commands to run.
	Commands map[string]string `name:"cmd" hide:"filegroup"`
	// Shell command to run for test targets.
	TestCommand string `name:"test_cmd"`
	// Per-configuration test commands to run.
	TestCommands map[string]string `name:"test_cmd"`
	// Represents the state of this build target (see below)
	state int32 `print:"false"`
	// True if this target is a binary (ie. runnable, will appear in plz-out/bin)
	IsBinary bool `name:"binary"`
	// True if this target is a test
	IsTest bool `name:"test"`
	// Indicates that the target can only be depended on by tests or other rules with this set.
	// Used to restrict non-deployable code and also affects coverage detection.
	TestOnly bool `name:"test_only"`
	// True if the build action is sandboxed.
	Sandbox bool
	// True if the test action is sandboxed.
	TestSandbox bool `name:"test_sandbox"`
	// True if the target is a test and has no output file.
	// Default is false, meaning all tests must produce test.results as output.
	NoTestOutput bool `name:"no_test_output"`
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
	// If true, the target is needed for a subinclude and therefore we will have to make sure its
	// outputs are available locally when built.
	NeededForSubinclude bool
	// Marks the target as a filegroup.
	IsFilegroup bool `print:"false"`
	// Marks the target as a hash_filegroup.
	IsHashFilegroup bool `print:"false"`
	// Marks the target as a remote_file.
	IsRemoteFile bool `print:"false"`
	// Marks that the target was added in a post-build function.
	AddedPostBuild bool `print:"false"`
	// If true, the interactive progress display will try to infer the target's progress
	// via some heuristics on its output.
	ShowProgress bool `name:"progress"`
	// If ShowProgress is true, this is used to store the current progress of the target.
	Progress float32 `print:"false"`
	// The results of this test target, if it is one.
	Results TestSuite `print:"false"`
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
	Provides map[string]BuildLabel
	// Stores the hash of this build rule before any post-build function is run.
	RuleHash []byte `name:"exported_deps"` // bit of a hack to call this exported_deps...
	// Tools that this rule will use, ie. other rules that it may use at build time which are not
	// copied into its source directory.
	Tools []BuildInput
	// Named tools, similar to named sources.
	namedTools map[string][]BuildInput `name:"tools"`
	// Target-specific environment passthroughs.
	PassEnv *[]string `name:"pass_env"`
	// Flakiness of test, ie. number of times we will rerun it before giving up. 1 is the default.
	Flakiness int `name:"flaky"`
	// Timeouts for build/test actions
	BuildTimeout time.Duration `name:"timeout"`
	TestTimeout  time.Duration `name:"test_timeout"`
	// Extra output files from the test.
	// These are in addition to the usual test.results output file.
	TestOutputs []string `name:"test_outputs"`
}

// BuildMetadata is temporary metadata that's stored around a build target - we don't
// generally persist it indefinitely.
type BuildMetadata struct {
	// Time the build began
	StartTime time.Time
	// Time it ended
	EndTime time.Time
	// Standard output & error
	Stdout, Stderr []byte
	// True if this represents a test run.
	Test bool
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
	declared BuildLabel     // the originally declared dependency
	deps     []*BuildTarget // list of actual deps
	resolved bool           // has the graph resolved it
	exported bool           // is it an exported dependency
	internal bool           // is it an internal dependency (that is not picked up implicitly by transitive searches)
	source   bool           // is it implicit because it's a source (not true if it's a dependency too)
	data     bool           // is it a data item for a test
}

// A BuildTargetState tracks the current state of this target in regard to whether it's built
// or not. Targets only move forwards through this (i.e. the state of a target only ever increases).
type BuildTargetState int32

// The available states for a target.
const (
	Inactive      BuildTargetState = iota // Target isn't used in current build
	Semiactive                            // Target would be active if we needed a build
	Active                                // Target is going to be used in current build
	Pending                               // Target is ready to be built but not yet started.
	Building                              // Target is currently being built
	Stopped                               // We stopped building the target because we'd gone as far as needed.
	Built                                 // Target has been successfully built
	Cached                                // Target has been retrieved from the cache
	Unchanged                             // Target has been built but hasn't changed since last build
	Reused                                // Outputs of previous build have been reused.
	BuiltRemotely                         // Target has been built but outputs are not necessarily local.
	Failed                                // Target failed for some reason
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
	} else if s == Failed {
		return "Failed"
	}
	return "Unknown"
}

// NewBuildTarget constructs & returns a new BuildTarget.
func NewBuildTarget(label BuildLabel) *BuildTarget {
	return &BuildTarget{
		Label:               label,
		state:               int32(Inactive),
		BuildingDescription: DefaultBuildingDescription,
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
	return path.Join(TmpDir, target.Label.Subrepo, target.Label.PackageName, target.Label.Name+buildDirSuffix)
}

// OutDir returns the output directory for this target, eg.
// //mickey/donald:goofy -> plz-out/gen/mickey/donald (or plz-out/bin if it's a binary)
func (target *BuildTarget) OutDir() string {
	if target.IsBinary {
		return path.Join(BinDir, target.Label.Subrepo, target.Label.PackageName)
	}
	return path.Join(GenDir, target.Label.Subrepo, target.Label.PackageName)
}

// TestDir returns the test directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy._test
// This is different to TmpDir so we run tests in a clean environment
// and to facilitate containerising tests.
func (target *BuildTarget) TestDir() string {
	return path.Join(TmpDir, target.Label.Subrepo, target.Label.PackageName, target.Label.Name+testDirSuffix)
}

// TestResultsFile returns the output results file for tests for this target.
func (target *BuildTarget) TestResultsFile() string {
	return path.Join(target.OutDir(), ".test_results_"+target.Label.Name)
}

// CoverageFile returns the output coverage file for tests for this target.
func (target *BuildTarget) CoverageFile() string {
	return path.Join(target.OutDir(), ".test_coverage_"+target.Label.Name)
}

// AllSourcePaths returns all the source paths for this target
func (target *BuildTarget) AllSourcePaths(graph *BuildGraph) []string {
	return target.allSourcePaths(graph, BuildInput.Paths)
}

// AllFullSourcePaths returns all the source paths for this target, with a leading
// plz-out/gen etc if appropriate.
func (target *BuildTarget) AllFullSourcePaths(graph *BuildGraph) []string {
	return target.allSourcePaths(graph, BuildInput.FullPaths)
}

// AllLocalSourcePaths returns the local part of all the source paths for this target,
// i.e. without this target's package in it.
func (target *BuildTarget) AllLocalSourcePaths(graph *BuildGraph) []string {
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

// DeclaredDependencies returns all the targets this target declared any kind of dependency on (including sources and tools).
func (target *BuildTarget) DeclaredDependencies() []BuildLabel {
	ret := make(BuildLabels, len(target.dependencies))
	for i, dep := range target.dependencies {
		ret[i] = dep.declared
	}
	sort.Sort(ret)
	return ret
}

// DeclaredDependenciesStrict returns the original declaration of this target's dependencies.
func (target *BuildTarget) DeclaredDependenciesStrict() []BuildLabel {
	ret := make(BuildLabels, 0, len(target.dependencies))
	for _, dep := range target.dependencies {
		if !dep.exported && !dep.source && !target.IsTool(dep.declared) {
			ret = append(ret, dep.declared)
		}
	}
	sort.Sort(ret)
	return ret
}

// Dependencies returns the resolved dependencies of this target.
func (target *BuildTarget) Dependencies() []*BuildTarget {
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

// BuildDependencies returns the build-time dependencies of this target (i.e. not data and not internal).
func (target *BuildTarget) BuildDependencies() []*BuildTarget {
	ret := make(BuildTargets, 0, len(target.dependencies))
	for _, deps := range target.dependencies {
		if !deps.data && !deps.internal {
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
	ret := make(BuildLabels, 0, len(target.dependencies))
	for _, info := range target.dependencies {
		if info.exported {
			ret = append(ret, info.declared)
		}
	}
	return ret
}

// DependenciesFor returns the dependencies that relate to a given label.
func (target *BuildTarget) DependenciesFor(label BuildLabel) []*BuildTarget {
	if info := target.dependencyInfo(label); info != nil {
		return info.deps
	} else if target.Label.Subrepo != "" && label.Subrepo == "" {
		// Can implicitly use the target's subrepo.
		label.Subrepo = target.Label.Subrepo
		return target.DependenciesFor(label)
	}
	return nil
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

// Outputs returns a slice of all the outputs of this rule.
func (target *BuildTarget) Outputs() []string {
	var ret []string
	if target.IsFilegroup && !target.IsHashFilegroup {
		ret = make([]string, 0, len(target.Sources))
		// Filegroups just re-output their inputs.
		for _, src := range target.Sources {
			if namedLabel, ok := src.(NamedOutputLabel); ok {
				// Bit of a hack, but this needs different treatment from either of the others.
				for _, dep := range target.DependenciesFor(namedLabel.BuildLabel) {
					ret = append(ret, dep.NamedOutputs(namedLabel.Output)...)
				}
			} else if label := src.nonOutputLabel(); label == nil {
				ret = append(ret, src.LocalPaths(nil)[0])
			} else {
				for _, dep := range target.DependenciesFor(*label) {
					ret = append(ret, dep.Outputs()...)
				}
			}
		}
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
		outs[i] = path.Join(outDir, out)
	}
	return outs
}

// NamedOutputs returns a slice of all the outputs of this rule with a given name.
// If the name is not declared by this rule it panics.
func (target *BuildTarget) NamedOutputs(name string) []string {
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
	if parseOutput == target.Label.PackageName {
		return parseOutput + ".out"
	} else if target.Label.PackageName == "" && target.HasSource(parseOutput) {
		// This also fixes the case where source and output are the same, which can happen
		// when we're in the root directory.
		return parseOutput + ".out"
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
	if label := source.nonOutputLabel(); label != nil {
		ret := []string{}
		for _, providedLabel := range graph.TargetOrDie(*label).ProvideFor(target) {
			ret = append(ret, f(providedLabel, graph)...)
		}
		return ret
	}
	return f(source, graph)
}

// allDepsBuilt returns true if all the dependencies of a target are built.
func (target *BuildTarget) allDepsBuilt() bool {
	if !target.allDependenciesResolved() {
		return false // Target still has some deps pending parse.
	}
	for _, deps := range target.dependencies {
		for _, dep := range deps.deps {
			if dep.State() < Built {
				return false
			}
		}
	}
	return true
}

// allDependenciesResolved returns true once all the dependencies of a target have been
// parsed and resolved to real targets.
func (target *BuildTarget) allDependenciesResolved() bool {
	for _, deps := range target.dependencies {
		if !deps.resolved {
			return false
		}
	}
	return true
}

// CanSee returns true if target can see the given dependency, or false if not.
func (target *BuildTarget) CanSee(state *BuildState, dep *BuildTarget) bool {
	return target.Label.CanSee(state, dep)
}

// CheckDependencyVisibility checks that all declared dependencies of this target are visible to it.
// Returns an error if not, or nil if all's well.
func (target *BuildTarget) CheckDependencyVisibility(state *BuildState) error {
	for _, d := range target.dependencies {
		dep := state.Graph.TargetOrDie(d.declared)
		if !target.CanSee(state, dep) {
			return fmt.Errorf("Target %s isn't visible to %s", dep.Label, target.Label)
		} else if dep.TestOnly && !(target.IsTest || target.TestOnly) {
			if target.Label.isExperimental(state) {
				log.Warning("Test-only restrictions suppressed for %s since %s is in the experimental tree", dep.Label, target.Label)
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

// CheckSecrets checks that this target's secrets are available.
// We run this check before building because we don't attempt to copy them, but any rule
// requiring them will presumably fail if they aren't available.
// Returns an error if any aren't.
func (target *BuildTarget) CheckSecrets() error {
	for _, secret := range target.AllSecrets() {
		if path := ExpandHomePath(secret); !PathExists(path) {
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
	return target.dependencyInfo(label) != nil
}

// hasResolvedDependency returns true if a particular dependency has been resolved to real targets yet.
func (target *BuildTarget) hasResolvedDependency(label BuildLabel) bool {
	info := target.dependencyInfo(label)
	return info != nil && info.resolved
}

// resolveDependency resolves a particular dependency on a target.
func (target *BuildTarget) resolveDependency(label BuildLabel, dep *BuildTarget) {
	info := target.dependencyInfo(label)
	if info == nil {
		target.dependencies = append(target.dependencies, depInfo{declared: label})
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
		if info.declared == label {
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
		if l == label {
			return true
		}
	}
	return label == "test" && target.IsTest
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
func (target *BuildTarget) AddProvide(language string, label BuildLabel) {
	if target.Provides == nil {
		target.Provides = map[string]BuildLabel{language: label}
	} else {
		target.Provides[language] = label
	}
}

// ProvideFor returns the build label that we'd provide for the given target.
func (target *BuildTarget) ProvideFor(other *BuildTarget) []BuildLabel {
	ret := []BuildLabel{}
	if target.Provides != nil && len(other.Requires) != 0 {
		// Never do this if the other target has a data or tool dependency on us.
		for _, data := range other.Data {
			if label := data.Label(); label != nil && *label == target.Label {
				return []BuildLabel{target.Label}
			}
		}
		if other.IsTool(target.Label) {
			return []BuildLabel{target.Label}
		}
		for _, require := range other.Requires {
			if label, present := target.Provides[require]; present {
				ret = append(ret, label)
			}
		}
		if len(ret) > 0 {
			return ret
		}
	}
	return []BuildLabel{target.Label}
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
	if label := source.Label(); label != nil {
		target.AddMaybeExportedDependency(*label, false, true, false)
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
		target.NamedSources = map[string][]BuildInput{name: target.addSource(nil, source)}
	} else {
		target.NamedSources[name] = target.addSource(target.NamedSources[name], source)
	}
}

// AddNamedSecret adds a secret to the target which is tagged with a particular name.
// These will be made available in the environment at runtime, with key-format "SECRETS_<NAME>".
func (target *BuildTarget) AddNamedSecret(name string, secret string) {
	if target.NamedSecrets == nil {
		target.NamedSecrets = map[string][]string{name: target.addSecret(nil, secret)}
	} else {
		target.NamedSecrets[name] = target.addSecret(target.NamedSecrets[name], secret)
	}
}

// AddTool adds a new tool to the target.
func (target *BuildTarget) AddTool(tool BuildInput) {
	target.Tools = append(target.Tools, tool)
	if label := tool.Label(); label != nil {
		target.AddDependency(*label)
	}
}

// AddDatum adds a new item of data to the target.
func (target *BuildTarget) AddDatum(datum BuildInput) {
	target.Data = append(target.Data, datum)
	if label := datum.Label(); label != nil {
		target.AddDependency(*label)
		target.dependencyInfo(*label).data = true
	}
}

// AddNamedTool adds a new tool to the target.
func (target *BuildTarget) AddNamedTool(name string, tool BuildInput) {
	if target.namedTools == nil {
		target.namedTools = map[string][]BuildInput{name: {tool}}
	} else {
		target.namedTools[name] = append(target.namedTools[name], tool)
	}
	if label := tool.Label(); label != nil {
		target.AddDependency(*label)
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
// Adding a general command is still done by simply setting the TestCommand member.
func (target *BuildTarget) AddTestCommand(config, command string) {
	if target.TestCommand != "" {
		panic(fmt.Sprintf("Adding named test command %s to %s, but it already has a general test command set", config, target.Label))
	} else if target.TestCommands == nil {
		target.TestCommands = map[string]string{config: command}
	} else {
		target.TestCommands[config] = command
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
	return target.getCommand(state, target.TestCommands, target.TestCommand)
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
	ret := target.Sources[:]
	if target.NamedSources != nil {
		keys := make([]string, 0, len(target.NamedSources))
		for k := range target.NamedSources {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ret = append(ret, target.NamedSources[k]...)
		}
	}
	return ret
}

// AllLocalSources returns all the "local" sources of this rule, i.e. all sources that are
// actually sources in the repo, not other rules or system srcs etc.
func (target *BuildTarget) AllLocalSources() []string {
	ret := []string{}
	for _, src := range target.AllSources() {
		if file, ok := src.(FileLabel); ok {
			ret = append(ret, file.Paths(nil)[0])
		}
	}
	return ret
}

// HasSource returns true if this target has the given file as a source (named or not, or data).
func (target *BuildTarget) HasSource(source string) bool {
	for _, src := range append(target.AllSources(), target.Data...) {
		if src.String() == source { // Comparison is a bit dodgy tbh
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

// AllTools returns all the tools for this rule in some canonical order.
func (target *BuildTarget) AllTools() []BuildInput {
	if target.namedTools == nil {
		return target.Tools // Leave them in input order, that's sufficiently consistent.
	}
	tools := make([]BuildInput, len(target.Tools), len(target.Tools)+len(target.namedTools)*2)
	copy(tools, target.Tools)
	for _, name := range target.ToolNames() {
		tools = append(tools, target.namedTools[name]...)
	}
	return tools
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

// AllData returns all the data paths for this target.
func (target *BuildTarget) AllData(graph *BuildGraph) []string {
	ret := make([]string, 0, len(target.Data))
	for _, datum := range target.Data {
		ret = append(ret, datum.Paths(graph)...)
	}
	return ret
}

// AddDependency adds a dependency to this target. It deduplicates against any existing deps.
func (target *BuildTarget) AddDependency(dep BuildLabel) {
	target.AddMaybeExportedDependency(dep, false, false, false)
}

// AddMaybeExportedDependency adds a dependency to this target which may be exported. It deduplicates against any existing deps.
func (target *BuildTarget) AddMaybeExportedDependency(dep BuildLabel, exported, source, internal bool) {
	if dep == target.Label {
		log.Fatalf("Attempted to add %s as a dependency of itself.\n", dep)
	}
	info := target.dependencyInfo(dep)
	if info == nil {
		target.dependencies = append(target.dependencies, depInfo{declared: dep, exported: exported, source: source, internal: internal})
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
		if t == tool {
			return true
		}
	}
	for _, tools := range target.namedTools {
		for _, t := range tools {
			if t == tool {
				return true
			}
		}
	}
	return false
}

// toolPath returns a path to this target when used as a tool.
func (target *BuildTarget) toolPath(abs bool) string {
	outputs := target.Outputs()
	ret := make([]string, len(outputs))
	for i, o := range outputs {
		if abs {
			ret[i] = path.Join(RepoRoot, target.OutDir(), o)
		} else {
			ret[i] = path.Join(target.Label.PackageName, o)
		}
	}
	return strings.Join(ret, " ")
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
	target.TestOutputs = target.insert(target.TestOutputs, output)
}

// AddNamedOutput adds a new output to the target under a named group.
// No attempt to deduplicate against unnamed outputs is currently made.
func (target *BuildTarget) AddNamedOutput(name, output string) {
	if target.namedOutputs == nil {
		target.namedOutputs = map[string][]string{name: target.insert(nil, output)}
		return
	}
	target.namedOutputs[name] = target.insert(target.namedOutputs[name], output)
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

// PostBuildOutputFileName returns the post-build output file for this target.
func (target *BuildTarget) PostBuildOutputFileName() string {
	return ".build_output_" + target.Label.Name
}

// NeedCoverage returns true if this target should output coverage during a test
// for a particular invocation.
func (target *BuildTarget) NeedCoverage(state *BuildState) bool {
	return state.NeedCoverage && !target.NoTestOutput && !target.HasAnyLabel(state.Config.Test.DisableCoverage)
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

// ShouldShowProgress returns true if the target should display progress.
// This is provided as a function to satisfy the process package.
func (target *BuildTarget) ShouldShowProgress() bool {
	return target.ShowProgress
}

// ProgressDescription returns a description of what the target is doing as it runs.
// This is provided as a function to satisfy the process package.
func (target *BuildTarget) ProgressDescription() string {
	if target.State() >= Built && target.IsTest {
		return "testing"
	}
	return target.BuildingDescription
}

// SetProgress sets the current progress of this target.
func (target *BuildTarget) SetProgress(progress float32) {
	target.Progress = progress
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
