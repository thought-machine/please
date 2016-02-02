package core

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// OutDir is the output directory for everything.
const OutDir string = "plz-out"
const tmpDir string = "plz-out/tmp"
const genDir string = "plz-out/gen"
const binDir string = "plz-out/bin"

// Representation of a build target and all information about it;
// its name, dependencies, build commands, etc.

type BuildTarget struct {
	// Identifier of this build target
	Label BuildLabel
	// Dependencies of this target
	Dependencies []*BuildTarget
	// Label dependencies of this target.
	// This is mostly used during the parse phase when we know the set of
	// declared labels, but don't necessarily have targets to associate
	// with them until we've parsed the relevant build files.
	DeclaredDependencies []BuildLabel
	// Dependencies that are 'exported' to consuming rules, ie. if something depends
	// on this rule, they get the exported dependencies as well.
	ExportedDependencies []BuildLabel
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
	// Optional labels applied to this rule. Used for including/excluding rules.
	Labels []string
	// Shell command to run.
	Command string
	// Shell command to run for test targets.
	TestCommand string
	// Represents the state of this build target (see below)
	state int32
	// True if this target is a binary (ie. runnable, will appear in plz-out/bin)
	IsBinary bool
	// True if this target is a test
	IsTest bool
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
	Tools []BuildLabel
	// Flakiness of test, ie. number of times we will rerun it before giving up. 0 is the default and
	// is interpreted the same way as 1 would be (ie. one run only).
	Flakiness int
	// Timeouts for build/test actions, in seconds.
	BuildTimeout int
	TestTimeout  int
	// Indication that we should skip caching this rule. This shouldn't be used as an out for
	// making targets indeterminate, it's a hint for rules like filegroup which simply symlink
	// their inputs - so it's faster to relink them than copying their contents.
	SkipCache bool
	// Indicates that the target can only be depended on by tests or other rules with this set.
	// Used to restrict non-deployable code and also affects coverage detection.
	TestOnly bool
	// Extra output files from the test.
	// These are in addition to the usual test.results output file.
	TestOutputs []string
	// Used internally to track how many dependencies we've resolved, ie. how many declared
	// dependencies have been mapped to actual dependencies. Once all are done this target is
	// ready to build.
	// TODO(pebers): unify this, Dependencies and DeclaredDependencies into one map.
	resolvedDependencies map[BuildLabel]bool
}

type BuildTargetState int32

const (
	Inactive   BuildTargetState = iota // Target isn't used in current build
	Semiactive BuildTargetState = iota // Target would be active if we needed a build
	Active     BuildTargetState = iota // Target is going to be used in current build
	Pending    BuildTargetState = iota // Target is ready to be built but not yet started.
	Building   BuildTargetState = iota // Target is currently being built
	Built      BuildTargetState = iota // Target has been successfully built
	Unchanged  BuildTargetState = iota // Target hasn't changed since last build
	Failed     BuildTargetState = iota // Target failed for some reason
)

type TestResults struct {
	NumTests         int // Total number of test cases in the test target.
	Passed           int // Number of tests that passed outright.
	Failed           int // Number of tests that failed.
	ExpectedFailures int // Number of tests that were expected to fail (counts as a pass, but displayed differently)
	Skipped          int // Number of tests skipped (also count as passes)
	Flakes           int // Number of failed attempts to run the test
	Failures         []TestFailure
	Passes           []string
	Output           string  // Stdout / stderr from the test.
	Cached           bool    // True if the test results were retrieved from cache
	Duration         float64 // Length of time this test took, in seconds.
}

type TestFailure struct {
	Name      string // Name of failed test
	Type      string // Type of failure, eg. type of exception raised
	Traceback string // Traceback
	Stdout    string // Standard output during test
	Stderr    string // Standard error during test
}

// Aggregates the given results into this one.
func (this *TestResults) Aggregate(that TestResults) {
	this.NumTests += that.NumTests
	this.Passed += that.Passed
	this.Failed += that.Failed
	this.ExpectedFailures += that.ExpectedFailures
	this.Skipped += that.Skipped
	this.Flakes += that.Flakes
	this.Failures = append(this.Failures, that.Failures...)
	this.Passes = append(this.Passes, that.Passes...)
	this.Duration += that.Duration
	// Output can't really be aggregated sensibly.
}

// Inputs to a build can be either a file in the local package or another build rule.
// All users care about is where they find them.
type BuildInput interface {
	// Returns a slice of paths to the files of this input.
	Paths(graph *BuildGraph) []string
	// As above, but includes the leading plz-out/gen directory.
	FullPaths(graph *BuildGraph) []string
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
	target.DeclaredDependencies = []BuildLabel{}
	target.state = int32(Inactive)
	target.Dependencies = []*BuildTarget{}
	target.resolvedDependencies = map[BuildLabel]bool{}
	target.IsBinary = false
	target.IsTest = false
	target.Command = "false"
	target.BuildingDescription = "Building..."
	return target
}

// Returns the temporary working directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy
// Note the extra subdirectory to keep rules separate from one another.
func (target *BuildTarget) TmpDir() string {
	return path.Join(tmpDir, target.Label.PackageName, target.Label.Name)
}

// Returns the output directory for this target, eg.
// //mickey/donald:goofy -> plz-out/gen/mickey/donald (or plz-out/bin if it's a binary)
func (target *BuildTarget) OutDir() string {
	if target.IsBinary {
		return path.Join(binDir, target.Label.PackageName)
	} else {
		return path.Join(genDir, target.Label.PackageName)
	}
}

// Returns the test directory for this target, eg.
// //mickey/donald:goofy -> plz-out/tmp/mickey/donald/goofy.test
// This is different to TmpDir so we run tests in a clean environment
// and to facilitate containerising tests.
func (target *BuildTarget) TestDir() string {
	return target.TmpDir() + ".test"
}

// Returns all the source paths for this target
func (target *BuildTarget) AllSourcePaths(graph *BuildGraph) []string {
	ret := make([]string, 0, len(target.Sources))
	for source := range target.AllSources() {
		for _, file := range source.Paths(graph) {
			ret = append(ret, file)
		}
	}
	return ret
}

// DeclaredOutputs returns the outputs from this target's original declaration.
// Hence it's similar to Outputs() but without the resolving of other rule names.
func (target *BuildTarget) DeclaredOutputs() []string {
	return target.outputs
}

// Outputs returns a slice of all the outputs of this rule.
// Recall that outputs can be defined as the name of another rule to indicate that
// a rule collects and re-outputs them; that is expanded here.
func (target *BuildTarget) Outputs() []string {
	ret := []string{}
	for _, out := range target.outputs {
		if LooksLikeABuildLabel(out) {
			label, file := ParseBuildFileLabel(out, target.Label.PackageName)
			if file != "" {
				ret = append(ret, file)
			} else {
				ret = append(ret, target.findOutputTarget(label, out)...)
			}
		} else {
			ret = append(ret, out)
		}
	}
	return ret
}

// findOutputTarget finds, among this target's dependencies, the target that outputs
// the given label, and returns its outputs.
func (target *BuildTarget) findOutputTarget(label BuildLabel, out string) []string {
	for _, dep := range target.Dependencies {
		if dep.Label == label {
			return dep.Outputs()
		}
	}
	panic(fmt.Sprintf("Target %s declares outputs of %s but they're not a resolved dependency of it", target.Label, out))
}

// Returns the source paths for a given set of sources.
func (target *BuildTarget) SourcePaths(graph *BuildGraph, sources []BuildInput) []string {
	ret := make([]string, 0, len(sources))
	for _, source := range sources {
		for _, file := range source.Paths(graph) {
			ret = append(ret, file)
		}
	}
	return ret
}

// allDepsBuilt returns true if all the dependencies of a target are built.
func (target *BuildTarget) allDepsBuilt() bool {
	if !target.allDependenciesResolved() {
		return false // Target still has some deps pending parse.
	}
	for _, dep := range target.Dependencies {
		if dep.State() < Built {
			return false
		}
	}
	return true
}

// allDependenciesResolved returns true once all the dependencies of a target have been
// parsed and resolved to real targets.
func (target *BuildTarget) allDependenciesResolved() bool {
	for _, resolved := range target.resolvedDependencies {
		if !resolved {
			return false
		}
	}
	return true
}

// CanSee returns true if target can see the given dependency, or false if not.
func (target *BuildTarget) CanSee(dep *BuildTarget) bool {
	// Targets are always visible to other targets in the same directory.
	if target.Label.PackageName == dep.Label.PackageName {
		return true
	}
	for _, vis := range dep.Visibility {
		if strings.HasPrefix(target.Label.PackageName, vis.PackageName) {
			// We're in the same package or a subpackage of this visibility spec
			if vis.IsAllSubpackages() {
				return true
			} else if target.Label.PackageName == vis.PackageName {
				if target.Label.Name == vis.Name || vis.IsAllTargets() {
					return true
				}
			}
		}
	}
	return false
}

// CheckDependencyVisibility checks that all dependencies of this target are visible to it.
// Returns an error if not, or nil if all's well.
func (target *BuildTarget) CheckDependencyVisibility() error {
	for _, dep := range target.Dependencies {
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
	for _, dep := range target.DeclaredDependencies {
		if dep == label {
			return true
		}
	}
	return false
}

// Checks if a target already has an exported dependency on this label.
func (target *BuildTarget) HasExportedDependency(label BuildLabel) bool {
	for _, dep := range target.ExportedDependencies {
		if dep == label {
			return true
		}
	}
	return false
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
	return false
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

// Handles the typical include/exclude logic for a target's labels; returns true if
// target has any include label and not an exclude one.
func (target *BuildTarget) ShouldInclude(include, exclude []string) bool {
	return (len(include) == 0 || target.HasAnyLabel(include)) && !target.HasAnyLabel(exclude) && !target.HasLabel("manual")
}

func (target *BuildTarget) AddProvide(language string, label BuildLabel) {
	if label == target.Label {
		target.addProvide(language, target.Label)
		return
	}
	for _, dep := range target.DeclaredDependencies {
		if dep == label {
			target.addProvide(language, label)
			return
		}
	}
	panic(fmt.Sprintf("Target %s must depend on %s in order to provide it", target.Label, label))
}

func (target *BuildTarget) addProvide(language string, label BuildLabel) {
	if target.Provides == nil {
		target.Provides = map[string]BuildLabel{language: label}
	} else {
		target.Provides[language] = label
	}
}

// Returns the build label that we'd provide for the given target.
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

func (target *BuildTarget) AddNamedSource(name string, source BuildInput) {
	if target.NamedSources == nil {
		target.NamedSources = map[string][]BuildInput{}
	}
	target.NamedSources[name] = append(target.NamedSources[name], source)
}

func (target *BuildTarget) AllSources() <-chan BuildInput {
	ch := make(chan BuildInput, 10)
	go func() {
		for _, source := range target.Sources {
			ch <- source
		}
		if target.NamedSources != nil {
			keys := make([]string, 0, len(target.NamedSources))
			for k := range target.NamedSources {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				for _, source := range target.NamedSources[k] {
					ch <- source
				}
			}
		}
		close(ch)
	}()
	return ch
}

func (target *BuildTarget) AddDependency(dep BuildLabel) {
	if dep == target.Label {
		log.Fatalf("Attempted to add %s as a dependency of itself.\n", dep)
	}
	if !target.HasDependency(dep) {
		target.DeclaredDependencies = append(target.DeclaredDependencies, dep)
		target.resolvedDependencies[dep] = false
	}
}

func (target *BuildTarget) AddExportedDependency(dep BuildLabel) {
	if !target.HasExportedDependency(dep) {
		target.ExportedDependencies = append(target.ExportedDependencies, dep)
	}
}

// IsTool returns true if the given build label is a dependency of this target.
func (target *BuildTarget) IsTool(tool BuildLabel) bool {
	for _, t := range target.Tools {
		if t == tool {
			return true
		}
	}
	return false
}

// AddOutput adds a new output to the target if it's not already there.
func (target *BuildTarget) AddOutput(output string) {
	for _, out := range target.outputs {
		if out == output {
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

// Sets one of the fields on the container settings by name.
func (target *BuildTarget) SetContainerSetting(name, value string) {
	if target.ContainerSettings == nil {
		target.ContainerSettings = &TargetContainerSettings{}
	}
	t := reflect.TypeOf(*target.ContainerSettings)
	for i := 0; i < t.NumField(); i++ {
		if strings.ToLower(t.Field(i).Name) == name {
			v := reflect.ValueOf(target.ContainerSettings)
			v.Elem().Field(i).SetString(value)
			return
		}
	}
	panic(fmt.Sprintf("Field %s isn't a valid container setting", name))
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

// Representation of a package, ie. the part of the system (one or more
// directories) covered by a single build file.
type Package struct {
	// Name of the package, ie. //spam/eggs
	Name string
	// Filename of the build file that defined this package
	Filename string
	// Subincluded build defs files that this package imported
	Subincludes []string
	// Targets contained within the package
	Targets map[string]*BuildTarget
	// Set of output files from rules.
	Outputs map[string]*BuildTarget
	// Protects access to above
	Mutex sync.Mutex
}

func NewPackage(name string) *Package {
	pkg := new(Package)
	pkg.Name = name
	pkg.Targets = map[string]*BuildTarget{}
	pkg.Outputs = map[string]*BuildTarget{}
	return pkg
}

func (pkg *Package) RegisterSubinclude(filename string) {
	// Ensure these are unique.
	for _, fn := range pkg.Subincludes {
		if fn == filename {
			return
		}
	}
	pkg.Subincludes = append(pkg.Subincludes, filename)
}

// RegisterOutput registers a new output file in the map.
// Dies if the file has already been registered.
func (pkg *Package) RegisterOutput(fileName string, target *BuildTarget) {
	pkg.Mutex.Lock()
	defer pkg.Mutex.Unlock()
	originalFileName := fileName
	if target.IsBinary {
		fileName = ":_bin_" + fileName // Add some arbitrary prefix so they don't clash.
	}
	if existing, present := pkg.Outputs[fileName]; present && existing != target {
		log.Fatalf("Rules %s and %s in %s both attempt to output the same file: %s\n",
			existing.Label, target.Label, pkg.Filename, originalFileName)
	}
	pkg.Outputs[fileName] = target
}
