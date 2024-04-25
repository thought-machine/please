package asp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// filegroupCommand is the command we put on filegroup rules.
const filegroupCommand = pyString("filegroup")

// textFileCommand is the command we put on text_file rules.
const textFileCommand = pyString("text_file")

const defaultFlakiness = 3

const (
	nameBuildRuleArgIdx = iota
	cmdBuildRuleArgIdx
	testCMDBuildRuleArgIdx
	debugCMDBuildRuleArgIdx
	srcsBuildRuleArgIdx
	dataBuildRuleArgIdx
	debugDataBuildRuleArgIdx
	outsBuildRuleArgIdx
	depsBuildRuleArgIdx
	exportedDepsBuildRuleArgIdx
	secretsBuildRuleArgIdx
	toolsBuildRuleArgIdx
	testToolsBuildRuleArgIdx
	debugToolsBuildRuleArgIdx
	labelsBuildRuleArgIdx
	visibilityBuildRuleArgIdx
	hashesBuildRuleArgIdx
	binaryBuildRuleArgIdx
	testBuildRuleArgIdx
	testOnlyBuildRuleArgIdx
	buildingDescriptionBuildRuleArgIdx
	needsTransitiveDepsBuildRuleArgIdx
	outputIsCompleteBuildRuleArgIdx
	sandboxBuildRuleArgIdx
	testSandboxBuildRuleArgIdx
	noTestOutputBuildRuleArgIdx
	flakyBuildRuleArgIdx
	buildTimeoutBuildRuleArgIdx
	testTimeoutBuildRuleArgIdx
	preBuildBuildRuleArgIdx
	postBuildBuildRuleArgIdx
	requiresBuildRuleArgIdx
	providesBuildRuleArgIdx
	licencesBuildRuleArgIdx
	testOutputsBuildRuleArgIdx
	systemSrcsBuildRuleArgIdx
	stampBuildRuleArgIdx
	tagBuildRuleArgIdx
	optionalOutsBuildRuleArgIdx
	progressBuildRuleArgIdx
	sizeBuildRuleArgIdx
	urlsBuildRuleArgIdx
	internalDepsBuildRuleArgIdx
	passEnvBuildRuleArgIdx
	localBuildRuleArgIdx
	outDirsBuildRuleArgIdx
	exitOnErrorArgIdx
	entryPointsArgIdx
	envArgIdx
	fileContentArgIdx
	subrepoArgIdx
)

// createTarget creates a new build target as part of build_rule().
func createTarget(s *scope, args []pyObject) *core.BuildTarget {
	isTruthy := func(i int) bool {
		return args[i] != nil && args[i] != None && args[i].IsTruthy()
	}
	name := string(args[nameBuildRuleArgIdx].(pyString))
	testCmd := args[testCMDBuildRuleArgIdx]
	test := isTruthy(testBuildRuleArgIdx)
	// A bunch of error checking first
	s.NAssert(name == "all", "'all' is a reserved build target name.")
	s.NAssert(name == "", "Target name is empty")
	s.NAssert(strings.ContainsRune(name, '/'), "/ is a reserved character in build target names")
	s.NAssert(strings.ContainsRune(name, ':'), ": is a reserved character in build target names")

	if tag := args[tagBuildRuleArgIdx]; tag != nil {
		if tagStr := string(tag.(pyString)); tagStr != "" {
			name = tagName(name, tagStr)
		}
	}
	label, err := core.TryNewBuildLabel(s.pkg.Name, name)
	s.Assert(err == nil, "Invalid build target name %s", name)
	label.Subrepo = s.pkg.SubrepoName

	target := core.NewBuildTarget(label)
	target.Subrepo = s.pkg.Subrepo
	target.IsBinary = isTruthy(binaryBuildRuleArgIdx)
	target.IsSubrepo = isTruthy(subrepoArgIdx)
	target.NeedsTransitiveDependencies = isTruthy(needsTransitiveDepsBuildRuleArgIdx)
	target.OutputIsComplete = isTruthy(outputIsCompleteBuildRuleArgIdx)
	target.Sandbox = isTruthy(sandboxBuildRuleArgIdx)
	target.TestOnly = test || isTruthy(testOnlyBuildRuleArgIdx)
	target.ShowProgress.Set(isTruthy(progressBuildRuleArgIdx))
	target.IsRemoteFile = isTruthy(urlsBuildRuleArgIdx)
	target.IsTextFile = args[cmdBuildRuleArgIdx] == textFileCommand
	target.Local = isTruthy(localBuildRuleArgIdx)
	target.ExitOnError = isTruthy(exitOnErrorArgIdx)
	for _, o := range asStringList(s, args[outDirsBuildRuleArgIdx], "output_dirs") {
		target.AddOutputDirectory(o)
	}

	var size *core.Size
	if args[sizeBuildRuleArgIdx] != None {
		name := string(args[sizeBuildRuleArgIdx].(pyString))
		size = mustSize(s, name)
		target.AddLabel(name)
	}
	if args[passEnvBuildRuleArgIdx] != None {
		l := asStringList(s, mustList(args[passEnvBuildRuleArgIdx]), "pass_env")
		target.PassEnv = &l
	}

	target.BuildTimeout = sizeAndTimeout(s, size, args[buildTimeoutBuildRuleArgIdx], s.state.Config.Build.Timeout)
	target.Stamp = isTruthy(stampBuildRuleArgIdx)
	target.IsFilegroup = args[cmdBuildRuleArgIdx] == filegroupCommand
	if desc := args[buildingDescriptionBuildRuleArgIdx]; desc != nil && desc != None {
		target.BuildingDescription = string(desc.(pyString))
	}
	if target.IsBinary {
		target.AddLabel("bin")
	}
	if target.IsRemoteFile {
		target.AddLabel("remote")
	}
	target.Command, target.Commands = decodeCommands(s, args[cmdBuildRuleArgIdx])
	if test {
		target.Test = new(core.TestFields)

		if flaky := args[flakyBuildRuleArgIdx]; flaky != nil {
			if flaky == True {
				target.Test.Flakiness = defaultFlakiness
				target.AddLabel("flaky") // Automatically label flaky tests
			} else if flaky == False {
				target.Test.Flakiness = 1
			} else if i, ok := flaky.(pyInt); ok {
				if int(i) <= 1 {
					target.Test.Flakiness = 1
				} else {
					target.Test.Flakiness = uint8(i)
					target.AddLabel("flaky")
				}
			}
		} else {
			target.Test.Flakiness = 1
		}
		if testCmd != nil && testCmd != None {
			target.Test.Command, target.Test.Commands = decodeCommands(s, args[testCMDBuildRuleArgIdx])
		}
		target.Test.Timeout = sizeAndTimeout(s, size, args[testTimeoutBuildRuleArgIdx], s.state.Config.Test.Timeout)
		target.Test.Sandbox = isTruthy(testSandboxBuildRuleArgIdx)
		target.Test.NoOutput = isTruthy(noTestOutputBuildRuleArgIdx)
	}

	if err := validateSandbox(s.state, target); err != nil {
		log.Fatal(err)
	}

	if s.state.Config.Build.Config == "dbg" {
		target.Debug = new(core.DebugFields)
		target.Debug.Command, _ = decodeCommands(s, args[debugCMDBuildRuleArgIdx])
	}
	return target
}

// validateSandbox ensures that the target isn't opting out of the build/test sandbox when it's not allowed to
func validateSandbox(state *core.BuildState, target *core.BuildTarget) error {
	if target.IsFilegroup || len(state.Config.Sandbox.ExcludeableTargets) == 0 {
		return nil
	}
	if !target.IsRemoteFile {
		if target.Sandbox && (target.Test == nil || target.Test.Sandbox) {
			return nil
		}
	}

	if target.Label.PackageName == "_please" {
		return nil
	}
	for _, whitelist := range state.Config.Sandbox.ExcludeableTargets {
		if whitelist.Matches(target.Label) {
			return nil
		}
	}
	for _, dir := range state.Config.Parse.ExperimentalDir {
		if strings.HasPrefix(target.Label.PackageName, dir) {
			return nil
		}
	}

	return fmt.Errorf("%v is not whitelisted to opt out of the sandbox", target)
}

// sizeAndTimeout handles the size and build/test timeout arguments.
func sizeAndTimeout(s *scope, size *core.Size, timeout pyObject, defaultTimeout cli.Duration) time.Duration {
	switch t := timeout.(type) {
	case pyInt:
		if t > 0 {
			return time.Duration(t) * time.Second
		}
	case pyString:
		return time.Duration(mustSize(s, string(t)).Timeout)
	}
	if size != nil {
		return time.Duration(size.Timeout)
	}
	return time.Duration(defaultTimeout)
}

// mustSize looks up a size by name. It panics if it cannot be found.
func mustSize(s *scope, name string) *core.Size {
	size, present := s.state.Config.Size[name]
	s.Assert(present, "Unknown size %s", name)
	return size
}

// decodeCommands takes a Python object and returns it as a string and a map; only one will be set.
func decodeCommands(s *scope, obj pyObject) (string, map[string]string) {
	if obj == nil || obj == None {
		return "", nil
	} else if cmd, ok := obj.(pyString); ok {
		return strings.TrimSpace(string(cmd)), nil
	}
	cmds, ok := asDict(obj)
	s.Assert(ok, "Unknown type for command [%s]", obj.Type())
	// Have to convert all the keys too
	m := make(map[string]string, len(cmds))
	for k, v := range cmds {
		if v != None {
			sv, ok := v.(pyString)
			s.Assert(ok, "Unknown type for command")
			m[k] = strings.TrimSpace(string(sv))
		}
	}
	return "", m
}

// populateTarget sets the assorted attributes on a build target.
func populateTarget(s *scope, t *core.BuildTarget, args []pyObject) {
	if t.IsRemoteFile {
		for _, url := range mustList(args[urlsBuildRuleArgIdx]) {
			t.AddSource(core.URLLabel(url.(pyString)))
		}
	} else if t.IsTextFile {
		t.FileContent = args[fileContentArgIdx].(pyString).String()
	}
	addMaybeNamed(s, "srcs", args[srcsBuildRuleArgIdx], t.AddSource, t.AddNamedSource, false, false)
	addMaybeNamedOrString(s, "tools", args[toolsBuildRuleArgIdx], t.AddTool, t.AddNamedTool, true, true)
	addMaybeNamed(s, "system_srcs", args[systemSrcsBuildRuleArgIdx], t.AddSource, nil, true, false)
	addMaybeNamed(s, "data", args[dataBuildRuleArgIdx], t.AddDatum, t.AddNamedDatum, false, false)
	addMaybeNamedOutput(s, "outs", args[outsBuildRuleArgIdx], t.AddOutput, t.AddNamedOutput, t, false)
	addMaybeNamedOutput(s, "optional_outs", args[optionalOutsBuildRuleArgIdx], t.AddOptionalOutput, nil, t, true)
	t.HintDependencies(depLen(args[depsBuildRuleArgIdx]) + depLen(args[exportedDepsBuildRuleArgIdx]) + depLen(args[internalDepsBuildRuleArgIdx]))
	addDependencies(s, "deps", args[depsBuildRuleArgIdx], t, false, false)
	addDependencies(s, "exported_deps", args[exportedDepsBuildRuleArgIdx], t, true, false)
	addDependencies(s, "internal_deps", args[internalDepsBuildRuleArgIdx], t, false, true)
	addStrings(s, "labels", args[labelsBuildRuleArgIdx], t.AddLabel)
	addStrings(s, "hashes", args[hashesBuildRuleArgIdx], t.AddHash)
	addStrings(s, "licences", args[licencesBuildRuleArgIdx], t.AddLicence)
	addStrings(s, "requires", args[requiresBuildRuleArgIdx], t.AddRequire)
	if vis, ok := asList(args[visibilityBuildRuleArgIdx]); ok && len(vis) != 0 {
		if v, ok := vis[0].(pyString); ok && v == "PUBLIC" {
			t.Visibility = core.WholeGraph
		} else {
			addStrings(s, "visibility", args[visibilityBuildRuleArgIdx], func(str string) {
				t.Visibility = append(t.Visibility, parseVisibility(s, str))
			})
		}
	}
	addEntryPoints(s, args[entryPointsArgIdx], t)
	addEnv(s, args[envArgIdx], t)
	addMaybeNamedSecret(s, "secrets", args[secretsBuildRuleArgIdx], t.AddSecret, t.AddNamedSecret, t, true)
	addProvides(s, "provides", args[providesBuildRuleArgIdx], t)
	if f := callbackFunction(s, "pre_build", args[preBuildBuildRuleArgIdx], 1, "argument"); f != nil {
		t.PreBuildFunction = &preBuildFunction{f: f, s: s}
	}
	if f := callbackFunction(s, "post_build", args[postBuildBuildRuleArgIdx], 2, "arguments"); f != nil {
		t.PostBuildFunction = &postBuildFunction{f: f, s: s}
	}

	if t.IsTest() {
		addMaybeNamedOrString(s, "test_tools", args[testToolsBuildRuleArgIdx], t.AddTestTool, t.AddNamedTestTool, true, true)
		addMaybeNamedOutput(s, "test_outputs", args[testOutputsBuildRuleArgIdx], t.AddTestOutput, nil, t, false)
	}

	if t.Debug != nil {
		addMaybeNamed(s, "debug_data", args[debugDataBuildRuleArgIdx], t.AddDebugDatum, t.AddDebugNamedDatum, false, false)
		addMaybeNamedOrString(s, "debug_tools", args[debugToolsBuildRuleArgIdx], t.AddDebugTool, t.AddNamedDebugTool, true, true)
	}
}

// depLen returns the length of a (potential) list
func depLen(obj pyObject) int {
	if l, ok := asList(obj); ok {
		return len(l)
	}
	return 0
}

// addEntryPoints adds entry points to a target
func addEntryPoints(s *scope, arg pyObject, target *core.BuildTarget) {
	entryPointsPy, ok := asDict(arg)
	s.Assert(ok, "entry_points must be a dict")

	for name, entryPointPy := range entryPointsPy {
		entryPoint, ok := entryPointPy.(pyString)
		s.Assert(ok, "Values of entry_points must be strings, found %v at key %v", entryPointPy.Type(), name)
		target.AddEntryPoint(name, string(entryPoint))
	}
}

// addEnv adds entry points to a target
func addEnv(s *scope, arg pyObject, target *core.BuildTarget) {
	envPy, ok := asDict(arg)
	s.Assert(ok, "env must be a dict")

	env := make(map[string]string, len(envPy))
	for name, val := range envPy {
		v, ok := val.(pyString)
		s.Assert(ok, "Values of env must be strings, found %v at key %v", val.Type(), name)
		env[name] = string(v)
	}

	target.Env = env
}

// addMaybeNamed adds inputs to a target, possibly in named groups.
func addMaybeNamed(s *scope, name string, obj pyObject, anon func(core.BuildInput), named func(string, core.BuildInput), systemAllowed, tool bool) {
	if obj == nil {
		return
	}
	if l, ok := asList(obj); ok {
		for _, li := range l {
			if bi := parseBuildInput(s, li, name, systemAllowed, tool); bi != nil {
				anon(bi)
			}
		}
	} else if d, ok := asDict(obj); ok {
		s.Assert(named != nil, "%s cannot be given as a dict", name)
		for k, v := range d {
			if v != None {
				if l, ok := asList(v); ok {
					for _, li := range l {
						if bi := parseBuildInput(s, li, name, systemAllowed, tool); bi != nil {
							named(k, bi)
						}
					}
					continue
				}
				if str, ok := asString(v); ok {
					if bi := parseBuildInput(s, str, name, systemAllowed, tool); bi != nil {
						named(k, bi)
					}
					continue
				}
				s.Assert(ok, "Values of %s must be a string or lists of strings", name)
			}
		}
	} else if obj != None {
		s.Assert(false, "Argument %s must be a list or dict, not %s", name, obj.Type())
	}
}

// addMaybeNamedOrString adds inputs to a target, possibly in named groups.
func addMaybeNamedOrString(s *scope, name string, obj pyObject, anon func(core.BuildInput), named func(string, core.BuildInput), systemAllowed, tool bool) {
	if obj == nil {
		return
	}
	if str, ok := asString(obj); ok {
		if bi := parseBuildInput(s, str, name, systemAllowed, tool); bi != nil {
			anon(bi)
		}
		return
	}
	addMaybeNamed(s, name, obj, anon, named, systemAllowed, tool)
}

// addMaybeNamedOutput adds outputs to a target, possibly in a named group
func addMaybeNamedOutput(s *scope, name string, obj pyObject, anon func(string), named func(string, string), t *core.BuildTarget, optional bool) {
	if obj == nil {
		return
	}
	if l, ok := asList(obj); ok {
		for _, li := range l {
			if li != None {
				out, ok := li.(pyString)
				s.Assert(ok, "outs must be strings")
				anon(string(out))
				if !optional || !strings.HasPrefix(string(out), "*") {
					s.pkg.MustRegisterOutput(s.state, string(out), t)
				}
			}
		}
	} else if d, ok := asDict(obj); ok {
		s.Assert(named != nil, "%s cannot be given as a dict", name)
		for k, v := range d {
			l, ok := asList(v)
			s.Assert(ok, "Values must be lists of strings")
			for _, li := range l {
				if li != None {
					out, ok := li.(pyString)
					s.Assert(ok, "outs must be strings")
					named(k, string(out))
					if !optional || !strings.HasPrefix(string(out), "*") {
						s.pkg.MustRegisterOutput(s.state, string(out), t)
					}
				}
			}
		}
	} else if obj != None {
		s.Assert(false, "Argument %s must be a list or dict, not %s", name, obj.Type())
	}
}

// addMaybeNamedSecret adds outputs to a target, possibly in a named group
func addMaybeNamedSecret(s *scope, name string, obj pyObject, anon func(string), named func(string, string), t *core.BuildTarget, optional bool) {
	validateSecret := func(secret string) {
		s.NAssert(strings.HasPrefix(secret, "//"),
			"Secret %s of %s cannot be a build label", secret, t.Label.Name)
		s.Assert(filepath.IsAbs(secret) || strings.HasPrefix(secret, "~"),
			"Secret '%s' of %s is not an absolute path", secret, t.Label.Name)
	}

	if obj == nil {
		return
	}
	if l, ok := asList(obj); ok {
		for _, li := range l {
			if li != None {
				out, ok := li.(pyString)
				s.Assert(ok, "secrets must be strings")
				validateSecret(string(out))
				anon(string(out))
			}
		}
	} else if d, ok := asDict(obj); ok {
		s.Assert(named != nil, "%s cannot be given as a dict", name)
		for k, v := range d {
			l, ok := asList(v)
			s.Assert(ok, "Values must be lists of strings")
			for _, li := range l {
				if li != None {
					out, ok := li.(pyString)
					s.Assert(ok, "outs must be strings")
					validateSecret(string(out))
					named(k, string(out))
				}
			}
		}
	} else if obj != None {
		s.Assert(false, "Argument %s must be a list or dict, not %s", name, obj.Type())
	}
}

// addDependencies adds dependencies to a target, which may or may not be exported.
func addDependencies(s *scope, name string, obj pyObject, target *core.BuildTarget, exported, internal bool) {
	addStrings(s, name, obj, func(str string) {
		if s.state.Config.Bazel.Compatibility && !core.LooksLikeABuildLabel(str) && !strings.HasPrefix(str, "@") {
			// *sigh*... Bazel seems to allow an implicit : on the start of dependencies
			str = ":" + str
		}
		target.AddMaybeExportedDependency(assertNotPseudoLabel(s, s.parseLabelInPackage(str, s.pkg)), exported, false, internal)
	})
}

// addStrings adds an arbitrary set of strings to the target (e.g. labels etc).
func addStrings(s *scope, name string, obj pyObject, f func(string)) {
	if obj != nil && obj != None {
		l, ok := asList(obj)
		if !ok {
			s.Error("Argument %s must be a list, not %s", name, obj.Type())
		}
		for _, li := range l {
			str, ok := li.(pyString)
			if !ok && li != None {
				s.Error("%s must be strings", name)
			}
			if str != "" && li != None {
				f(string(str))
			}
		}
	}
}

// addProvides adds a set of provides to the target, which is a dict of string -> label
func addProvides(s *scope, name string, obj pyObject, t *core.BuildTarget) {
	if obj != nil && obj != None {
		d, ok := asDict(obj)
		s.Assert(ok, "Argument %s must be a dict, not %s, %v", name, obj.Type(), obj)
		for k, v := range d {
			if str, ok := v.(pyString); ok {
				t.AddProvide(k, []core.BuildLabel{assertNotPseudoLabel(s, s.parseLabelInPackage(string(str), s.pkg))})
				continue
			}
			if list, ok := v.(pyList); ok {
				ls := make([]core.BuildLabel, len(list))
				for i, v := range list {
					str, ok := v.(pyString)
					s.Assert(ok, "%s values must be strings or lists of strings", name)

					ls[i] = assertNotPseudoLabel(s, s.parseLabelInPackage(string(str), s.pkg))
				}
				t.AddProvide(k, ls)
				continue
			}
			s.Error("%s values must be strings or lists of strings", name)
		}
	}
}

// parseVisibility converts a visibility string to a build label.
// Mostly they are just build labels but other things are allowed too (e.g. "PUBLIC").
func parseVisibility(s *scope, vis string) core.BuildLabel {
	if vis == "PUBLIC" || (s.state.Config.Bazel.Compatibility && vis == "//visibility:public") {
		return core.WholeGraph[0]
	}
	l := s.parseLabelInPackage(vis, s.pkg)
	if s.state.Config.Bazel.Compatibility {
		// Bazel has a couple of special aliases for this stuff.
		if l.Name == "__pkg__" {
			l.Name = "all"
		} else if l.Name == "__subpackages__" {
			l.Name = "..."
		}
	}
	return l
}

func parseBuildInput(s *scope, in pyObject, name string, systemAllowed, tool bool) core.BuildInput {
	src, ok := in.(pyString)
	if !ok {
		s.Assert(in == None, "Items in %s must be strings", name)
		return nil
	}
	return parseSource(s, string(src), systemAllowed, tool)
}

// parseSource parses an incoming source label as either a file or a build label.
// Identifies if the file is owned by this package and returns an error if not.
func parseSource(s *scope, src string, systemAllowed, tool bool) core.BuildInput {
	if core.LooksLikeABuildLabel(src) {
		pkg := s.pkg
		if tool && s.pkg.Subrepo != nil && s.pkg.Subrepo.IsCrossCompile {
			// Tools should be parsed with the host OS and arch
			pkg = &core.Package{
				Name: pkg.Name,
			}
		}
		label := s.parseAnnotatedLabelInPackage(src, pkg)
		if l, ok := label.Label(); ok {
			assertNotPseudoLabel(s, l)
		}
		return label
	}
	s.Assert(src != "", "Empty source path")
	s.Assert(!strings.Contains(src, "../"), "%s is an invalid path; build target paths can't contain ../", src)
	if filepath.IsAbs(src) || src[0] == '~' {
		s.Assert(systemAllowed, "%s is an absolute path; that's not allowed", src)
		return core.SystemFileLabel{Path: strings.TrimRight(src, "/")}
	} else if tool {
		// "go" as a source is interpreted as a file, as a tool it's interpreted as something on the PATH.
		return core.SystemPathLabel{Name: src, Path: s.state.Config.Path()}
	}
	src = strings.TrimPrefix(src, "./")
	return core.NewFileLabel(src, s.pkg)
}

// assertNotPsudolabel checks that the given build label is not a pseudo-label.
// These are disallowed in (nearly) all contexts.
func assertNotPseudoLabel(s *scope, label core.BuildLabel) core.BuildLabel {
	s.NAssert(label.IsAllTargets(), ":all labels are not permitted here")
	s.NAssert(label.IsAllSubpackages(), "... labels are not permitted here")
	return label
}

// callbackFunction extracts a pre- or post-build function for a target.
func callbackFunction(s *scope, name string, obj pyObject, requiredArguments int, arguments string) *pyFunc {
	if obj != nil && obj != None {
		f := obj.(*pyFunc)
		s.Assert(len(f.args) == requiredArguments, "%s callbacks must take exactly %d %s (%s takes %d)", name, requiredArguments, arguments, f.name, len(f.args))
		return f
	}
	return nil
}

// A preBuildFunction implements the core.PreBuildFunction interface
type preBuildFunction struct {
	f *pyFunc
	s *scope
}

func (f *preBuildFunction) Call(target *core.BuildTarget) error {
	s := f.f.scope.NewPackagedScope(f.f.scope.state.Graph.PackageOrDie(target.Label), f.f.scope.mode, 1)
	s.config = f.s.config
	s.Set("CONFIG", f.s.config)
	s.Callback = true
	s.Set(f.f.args[0], pyString(target.Label.Name))
	_, err := s.interpreter.interpretStatements(s, f.f.code)
	return annotateCallbackError(s, target, err)
}

func (f *preBuildFunction) String() string {
	return f.f.String()
}

// A postBuildFunction implements the core.PostBuildFunction interface
type postBuildFunction struct {
	f *pyFunc
	s *scope
}

func (f *postBuildFunction) Call(target *core.BuildTarget, output string) error {
	s := f.f.scope.NewPackagedScope(f.f.scope.state.Graph.PackageOrDie(target.Label), f.f.scope.mode, 2)
	s.config = f.s.config
	s.Set("CONFIG", f.s.config)
	s.Callback = true
	s.Set(f.f.args[0], pyString(target.Label.Name))
	s.Set(f.f.args[1], fromStringList(strings.Split(strings.TrimSpace(output), "\n")))
	_, err := s.interpreter.interpretStatements(s, f.f.code)
	return annotateCallbackError(s, target, err)
}

func (f *postBuildFunction) String() string {
	return f.f.String()
}

// annotateCallbackError adds some information to an error on failure about where it was in the file.
func annotateCallbackError(s *scope, target *core.BuildTarget, err error) error {
	if err == nil {
		return nil
	}
	// Something went wrong, find the BUILD file and attach some info.
	pkg := s.state.Graph.PackageByLabel(target.Label)
	f, _ := os.Open(pkg.Filename)
	return s.interpreter.parser.annotate(err, f)
}

// asList converts an object to a pyList, accounting for frozen lists.
func asList(obj pyObject) (pyList, bool) {
	if l, ok := obj.(pyList); ok {
		return l, true
	} else if l, ok := obj.(pyFrozenList); ok {
		return l.pyList, true
	}
	return nil, false
}

// mustList is like asList but returns an empty list if the object isn't a list.
func mustList(obj pyObject) pyList {
	if l, ok := asList(obj); ok {
		return l
	}
	return pyList{}
}

// asDict converts an object to a pyDict, accounting for frozen dicts.
func asDict(obj pyObject) (pyDict, bool) {
	if d, ok := obj.(pyDict); ok {
		return d, true
	} else if d, ok := obj.(pyFrozenDict); ok {
		return d.pyDict, true
	}
	return nil, false
}

// asString converts an object to a pyString
func asString(obj pyObject) (pyString, bool) {
	if s, ok := obj.(pyString); ok {
		return s, true
	}
	return "", false
}
