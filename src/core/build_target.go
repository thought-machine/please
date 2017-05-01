package core

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// OutDir is the output directory for everything.
const OutDir string = "plz-out"
const TmpDir string = "plz-out/tmp"
const GenDir string = "plz-out/gen"
const BinDir string = "plz-out/bin"

// Default when this isn't otherwise specified.
const DefaultBuildingDescription = "Building..."

// Suffixes for temporary directories
const buildDirSuffix = "._build"
const testDirSuffix = "._test"

// Representation of a build target and all information about it;
// its name, dependencies, build commands, etc.

type BuildTarget struct {
	// Identifier of this build target
	Label BuildLabel
	// Dependencies of this target.
	// Maps the original declaration to whatever dependencies actually got attached,
	// which may be more than one in some cases. Also contains info about exporting etc.
	dependencies []depInfo
	// List of build target patterns that can use this build target.
	Visibility []BuildLabel
	// Source files of this rule. Can refer to build rules themselves.
	Sources []BuildInput
	// Named source files of this rule; as above but identified by name.
	NamedSources map[string][]BuildInput
	// Data files of this rule. Similar to sources but used at runtime, typically by tests.
	Data []BuildInput
	// Output files of this rule. All are paths relative to this package.
	outputs []string
	// Optional output files of this rule. Same as outs but aren't required to be produced always.
	// Can be glob patterns.
	OptionalOutputs []string
	// Optional labels applied to this rule. Used for including/excluding rules.
	Labels []string
	// Shell command to run.
	Command string
	// Per-configuration shell commands to run.
	Commands map[string]string
	// Shell command to run for test targets.
	TestCommand string
	// Per-configuration test commands to run.
	TestCommands map[string]string
	// Represents the state of this build target (see below)
	state int32
	// True if this target is a binary (ie. runnable, will appear in plz-out/bin)
	IsBinary bool
	// True if this target is a test
	IsTest bool
	// Indicates that the target can only be depended on by tests or other rules with this set.
	// Used to restrict non-deployable code and also affects coverage detection.
	TestOnly bool
	// True if we're going to containerise the test.
	Containerise bool
	// True if the target is a test and has no output file.
	// Default is false, meaning all tests must produce test.results as output.
	NoTestOutput bool
	// True if this target needs access to its transitive dependencies to build.
	// This would be false for most 'normal' genrules but true for eg. compiler steps
	// that need to build in everything.
	NeedsTransitiveDependencies bool
	// True if this target blocks recursive exploring for transitive dependencies.
	// This is typically false for _library rules which aren't complete, and true
	// for _binary rules which normally are, and genrules where you don't care about
	// the inputs, only whatever they were turned into.
	OutputIsComplete bool
	// If true, the rule is given an env var at build time that contains the hash of its
	// transitive dependencies, which can be used to identify the output in a predictable way.
	Stamp bool
	// Marks the target as a filegroup.
	IsFilegroup bool
	// Containerisation settings that override the defaults.
	ContainerSettings *TargetContainerSettings
	// Results of test, if it is one
	Results TestResults
	// Description displayed while the command is building.
	// Default is just "Building" but it can be customised.
	BuildingDescription string
	// Acceptable hashes of the outputs of this rule. If the output doesn't match any of these
	// it's an error at build time. Can be used to validate third-party deps.
	Hashes []string
	// Licences that this target is subject to.
	Licences []string
	// Python functions to call before / after target is built. Allows deferred manipulation of the
	// build graph.
	PreBuildFunction, PostBuildFunction uintptr
	// Hash of the function's bytecode. Used for incrementality.
	// TODO(pebers): unify with RuleHash maybe? seems wasteful to store these separately.
	PreBuildHash, PostBuildHash []byte
	// Languages this rule requires. These are an arbitrary set and the only meaning is that they
	// correspond to entries in Provides; if rules match up then it allows choosing a specific
	// dependency (consider eg. code generated from protobufs; this mechanism allows us to expose
	// one rule but only compile the appropriate code for each library that consumes it).
	Requires []string
	// Dependent rules this rule provides for each language. Matches up to Requires as described above.
	Provides map[string]BuildLabel
	// Stores the hash of this build rule before any post-build function is run.
	RuleHash []byte
	// Tools that this rule will use, ie. other rules that it may use at build time which are not
	// copied into its source directory.
	Tools []BuildInput
	// Flakiness of test, ie. number of times we will rerun it before giving up. 0 is the default and
	// is interpreted the same way as 1 would be (ie. one run only).
	Flakiness int
	// Timeouts for build/test actions
	BuildTimeout time.Duration
	TestTimeout  time.Duration
	// Extra output files from the test.
	// These are in addition to the usual test.results output file.
	TestOutputs []string
}

type depInfo struct {
	declared BuildLabel     // the originally declared dependency
	deps     []*BuildTarget // list of actual deps
	resolved bool           // has the graph resolved it
	exported bool           // is it an exported dependency
}

type BuildTargetState int32

const (
	Inactive   BuildTargetState = iota // Target isn't used in current build
	Semiactive                         // Target would be active if we needed a build
	Active                             // Target is going to be used in current build
	Pending                            // Target is ready to be built but not yet started.
	Building                           // Target is currently being built
	Stopped                            // We stopped building the target because we'd gone as far as needed.
	Built                              // Target has been successfully built
	Cached                             // Target has been retrieved from the cache
	Unchanged                          // Target has been built but hasn't changed since last build
	Reused                             // Outputs of previous build have been reused.
	Failed                             // Target failed for some reason
)

// String implements the fmt.Stringer interface.
// TODO(pebers): Convert this to use go generate / stringer.
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

// Inputs to a build can be either a file in the local package or another build rule.
// All users care about is where they find them.
type BuildInput interface {
	// Returns a slice of paths to the files of this input.
	Paths(graph *BuildGraph) []string
	// As above, but includes the leading plz-out/gen directory.
	FullPaths(graph *BuildGraph) []string
	// Paths within the local package
	LocalPaths(graph *BuildGraph) []string
	// Returns the build label associated with this input, or nil if it doesn't have one (eg. it's just a file).
	Label() *BuildLabel
	// Returns a string representation of this input
	String() string
}

// Settings controlling containerisation for a particular target.
type TargetContainerSettings struct {
	// Image to use for this test
	DockerImage string
	// Username / Uid to run as
	DockerUser string
	// Extra arguments to pass to 'docker run'
	DockerRunArgs string
}

func NewBuildTarget(label BuildLabel) *BuildTarget {
	target := new(BuildTarget)
	target.Label = label
	target.state = int32(Inactive)
	target.IsBinary = false
	target.IsTest = false
	target.BuildingDescription = DefaultBuildingDescription
	return target
}

// TmpDir returns the temporary working directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy._build
// Note the extra subdirectory to keep rules separate from one another, and the .build suffix
// to attempt to keep rules from duplicating the names of sub-packages; obviously that is not
// 100% reliable but we don't have a better solution right now.
func (target *BuildTarget) TmpDir() string {
	return path.Join(TmpDir, target.Label.PackageName, target.Label.Name+buildDirSuffix)
}

// Returns the output directory for this target, eg.
// //mickey/donald:goofy -> plz-out/gen/mickey/donald (or plz-out/bin if it's a binary)
func (target *BuildTarget) OutDir() string {
	if target.IsBinary {
		return path.Join(BinDir, target.Label.PackageName)
	} else {
		return path.Join(GenDir, target.Label.PackageName)
	}
}

// Returns the test directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy._test
// This is different to TmpDir so we run tests in a clean environment
// and to facilitate containerising tests.
func (target *BuildTarget) TestDir() string {
	return path.Join(TmpDir, target.Label.PackageName, target.Label.Name+testDirSuffix)
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

// DeclaredDependencies returns the original declaration of this target's dependencies.
func (target *BuildTarget) DeclaredDependencies() []BuildLabel {
	ret := make(BuildLabels, 0, len(target.dependencies))
	for _, dep := range target.dependencies {
		ret = append(ret, dep.declared)
	}
	sort.Sort(ret)
	return ret
}

// Dependencies returns the resolved dependencies of this target.
func (target *BuildTarget) Dependencies() []*BuildTarget {
	ret := make(BuildTargets, 0, len(target.dependencies))
	for _, deps := range target.dependencies {
		for _, dep := range deps.deps {
			// N.B. Include any exported dependencies of this guy too.
			ret = append(ret, dep)
			ret = append(ret, dep.transitiveExportedDependencies()...)
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

// transitiveExportedDependencies returns the transitive set of exported dependencies of this target.
func (target *BuildTarget) transitiveExportedDependencies() []*BuildTarget {
	var ret []*BuildTarget
	for _, info := range target.dependencies {
		if info.exported {
			for _, dep := range info.deps {
				ret = append(ret, dep)
				ret = append(ret, dep.transitiveExportedDependencies()...)
			}
		}
	}
	return ret
}

// DependenciesFor returns the dependencies that relate to a given label.
func (target *BuildTarget) DependenciesFor(label BuildLabel) []*BuildTarget {
	info := target.dependencyInfo(label)
	if info != nil {
		return info.deps
	}
	return nil
}

// DeclaredOutputs returns the outputs from this target's original declaration.
// Hence it's similar to Outputs() but without the resolving of other rule names.
func (target *BuildTarget) DeclaredOutputs() []string {
	return target.outputs
}

// Outputs returns a slice of all the outputs of this rule.
func (target *BuildTarget) Outputs() []string {
	var ret []string
	if target.IsFilegroup {
		ret = make([]string, 0, len(target.Sources))
		// Filegroups just re-output their inputs.
		for _, src := range target.Sources {
			if label := src.Label(); label == nil {
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
	sort.Strings(ret)
	return ret
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
	if label := source.Label(); label != nil {
		ret := []string{}
		for _, providedLabel := range graph.TargetOrDie(*label).ProvideFor(target) {
			for _, file := range f(providedLabel, graph) {
				ret = append(ret, file)
			}
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

// isExperimental returns true if the given target is in the "experimental" tree
func isExperimental(target *BuildTarget) bool {
	return State.experimentalLabel.PackageName != "" && State.experimentalLabel.Includes(target.Label)
}

// CanSee returns true if target can see the given dependency, or false if not.
func (target *BuildTarget) CanSee(dep *BuildTarget) bool {
	// Targets are always visible to other targets in the same directory.
	if target.Label.PackageName == dep.Label.PackageName {
		return true
	}
	if isExperimental(dep) && !isExperimental(target) {
		log.Error("Target %s cannot depend on experimental target %s", target.Label, dep.Label)
		return false
	}
	for _, vis := range dep.Visibility {
		if vis.Includes(target.Label.Parent()) {
			return true
		}
	}
	if isExperimental(target) {
		log.Warning("Visibility restrictions suppressed for %s since %s is in the experimental tree", dep.Label, target.Label)
		return true
	}
	return false
}

// CheckDependencyVisibility checks that all declared dependencies of this target are visible to it.
// Returns an error if not, or nil if all's well.
func (target *BuildTarget) CheckDependencyVisibility(graph *BuildGraph) error {
	for _, d := range target.dependencies {
		dep := graph.TargetOrDie(d.declared)
		if !target.CanSee(dep) {
			return fmt.Errorf("Target %s isn't visible to %s", dep.Label, target.Label)
		} else if dep.TestOnly && !(target.IsTest || target.TestOnly) {
			return fmt.Errorf("Target %s can't depend on %s, it's marked test_only", target.Label, dep.Label)
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

// HasAnyLabel returns true if target has any of these labels.
func (target *BuildTarget) HasAnyLabel(labels []string) bool {
	for _, label := range labels {
		if target.HasLabel(label) {
			return true
		}
	}
	return false
}

// ShouldInclude handles the typical include/exclude logic for a target's labels; returns true if
// target has any include label and not an exclude one.
func (target *BuildTarget) ShouldInclude(include, exclude []string) bool {
	return (len(include) == 0 || target.HasAnyLabel(include)) && !target.HasAnyLabel(exclude) && !target.HasLabel("manual")
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
	if target.Provides != nil {
		// Never do this if the other target has a data dependency on us.
		for _, data := range other.Data {
			if label := data.Label(); label != nil && *label == target.Label {
				return []BuildLabel{target.Label}
			}
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
		target.AddDependency(*label)
	}
	return append(sources, source)
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
func (target *BuildTarget) GetCommand() string {
	return target.getCommand(target.Commands, target.Command)
}

// GetCommandConfig returns the command we should use to build this target for the given config.
func (target *BuildTarget) GetCommandConfig(config string) string {
	if config == "" {
		return target.Command
	}
	return target.Commands[config]
}

// GetTestCommand returns the command we should use to test this target for the current config.
func (target *BuildTarget) GetTestCommand() string {
	return target.getCommand(target.TestCommands, target.TestCommand)
}

func (target *BuildTarget) getCommand(commands map[string]string, singleCommand string) string {
	if commands == nil {
		return singleCommand
	} else if command, present := commands[State.Config.Build.Config]; present {
		return command // Has command for current config, good
	} else if command, present := commands[State.Config.Build.FallbackConfig]; present {
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
		target.Label, State.Config.Build.Config, State.Config.Build.FallbackConfig, highestConfig)
	return highestCommand
}

// AllSources returns all the sources of this rule.
func (target *BuildTarget) AllSources() []BuildInput {
	ret := make([]BuildInput, 0, len(target.Sources))
	for _, source := range target.Sources {
		ret = append(ret, source)
	}
	if target.NamedSources != nil {
		keys := make([]string, 0, len(target.NamedSources))
		for k := range target.NamedSources {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, source := range target.NamedSources[k] {
				ret = append(ret, source)
			}
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

// HasSource returns true if this target has the given file as a source (named or not).
func (target *BuildTarget) HasSource(source string) bool {
	for _, src := range target.AllSources() {
		if src.String() == source { // Comparison is a bit dodgy tbh
			return true
		}
	}
	return false
}

// AddDependency adds a dependency to this target. It deduplicates against any existing deps.
func (target *BuildTarget) AddDependency(dep BuildLabel) {
	target.AddMaybeExportedDependency(dep, false)
}

// AddMaybeExportedDependency adds a dependency to this target which may be exported. It deduplicates against any existing deps.
func (target *BuildTarget) AddMaybeExportedDependency(dep BuildLabel, exported bool) {
	if dep == target.Label {
		log.Fatalf("Attempted to add %s as a dependency of itself.\n", dep)
	}
	info := target.dependencyInfo(dep)
	if info == nil {
		target.dependencies = append(target.dependencies, depInfo{declared: dep, exported: exported})
	} else if exported {
		info.exported = exported
	}
}

// IsTool returns true if the given build label is a tool used by this target.
func (target *BuildTarget) IsTool(tool BuildLabel) bool {
	for _, t := range target.Tools {
		if t == tool {
			return true
		}
	}
	return false
}

// toolPath returns a path to this target when used as a tool.
func (target *BuildTarget) toolPath() string {
	outputs := target.Outputs()
	ret := make([]string, len(outputs))
	for i, o := range outputs {
		ret[i], _ = filepath.Abs(path.Join(target.OutDir(), o))
	}
	return strings.Join(ret, " ")
}

// AddOutput adds a new output to the target if it's not already there.
func (target *BuildTarget) AddOutput(output string) {
	for i, out := range target.outputs {
		if out == output {
			return
		} else if out > output {
			// Insert in sorted order, with an attempt to be efficient.
			target.outputs = append(target.outputs, "")
			copy(target.outputs[i+1:], target.outputs[i:])
			target.outputs[i] = output
			return
		}
	}
	target.outputs = append(target.outputs, output)
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

// SetContainerSetting sets one of the fields on the container settings by name.
func (target *BuildTarget) SetContainerSetting(name, value string) error {
	if target.ContainerSettings == nil {
		target.ContainerSettings = &TargetContainerSettings{}
	}
	t := reflect.TypeOf(*target.ContainerSettings)
	for i := 0; i < t.NumField(); i++ {
		if strings.ToLower(t.Field(i).Name) == name {
			v := reflect.ValueOf(target.ContainerSettings)
			v.Elem().Field(i).SetString(value)
			return nil
		}
	}
	return fmt.Errorf("Field %s isn't a valid container setting", name)
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

// Make slices of these guys sortable.
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
