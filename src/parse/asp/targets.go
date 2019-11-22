package asp

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// filegroupCommand is the command we put on filegroup rules.
const filegroupCommand = pyString("filegroup")

// hashFilegroupCommand is similarly the command for hash_filegroup rules.
const hashFilegroupCommand = pyString("hash_filegroup")

// createTarget creates a new build target as part of build_rule().
func createTarget(s *scope, args []pyObject) *core.BuildTarget {
	isTruthy := func(i int) bool {
		return args[i] != nil && args[i] != None && (args[i] == &True || args[i].IsTruthy())
	}
	name := string(args[0].(pyString))
	testCmd := args[2]
	test := isTruthy(14)
	// A bunch of error checking first
	s.NAssert(name == "all", "'all' is a reserved build target name.")
	s.NAssert(name == "", "Target name is empty")
	s.NAssert(strings.ContainsRune(name, '/'), "/ is a reserved character in build target names")
	s.NAssert(strings.ContainsRune(name, ':'), ": is a reserved character in build target names")

	if tag := args[34]; tag != nil {
		if tagStr := string(tag.(pyString)); tagStr != "" {
			name = tagName(name, tagStr)
		}
	}
	label, err := core.TryNewBuildLabel(s.pkg.Name, name)
	s.Assert(err == nil, "Invalid build target name %s", name)
	label.Subrepo = s.pkg.SubrepoName

	target := core.NewBuildTarget(label)
	target.Subrepo = s.pkg.Subrepo
	target.IsBinary = isTruthy(13)
	target.IsTest = test
	target.NeedsTransitiveDependencies = isTruthy(17)
	target.OutputIsComplete = isTruthy(18)
	target.Sandbox = isTruthy(20)
	target.TestOnly = test || isTruthy(15)
	target.ShowProgress = isTruthy(36)
	target.IsRemoteFile = isTruthy(38)
	target.Local = isTruthy(41)

	var size *core.Size
	if args[37] != None {
		name := string(args[37].(pyString))
		size = mustSize(s, name)
		target.AddLabel(name)
	}
	if args[40] != None {
		l := asStringList(s, args[40].(pyList), "pass_env")
		target.PassEnv = &l
	}

	target.BuildTimeout = sizeAndTimeout(s, size, args[24], s.state.Config.Build.Timeout)
	target.Stamp = isTruthy(33)
	target.IsHashFilegroup = args[1] == hashFilegroupCommand
	target.IsFilegroup = args[1] == filegroupCommand || target.IsHashFilegroup
	if desc := args[16]; desc != nil && desc != None {
		target.BuildingDescription = string(desc.(pyString))
	}
	if target.IsBinary {
		target.AddLabel("bin")
	}
	target.Command, target.Commands = decodeCommands(s, args[1])
	if test {
		if flaky := args[23]; flaky != nil {
			if flaky == True {
				target.Flakiness = 3
				target.AddLabel("flaky") // Automatically label flaky tests
			} else if flaky == False {
				target.Flakiness = 1
			} else if i, ok := flaky.(pyInt); ok {
				if int(i) <= 1 {
					target.Flakiness = 1
				} else {
					target.Flakiness = int(i)
					target.AddLabel("flaky")
				}
			}
		} else {
			target.Flakiness = 1
		}
		if testCmd != nil && testCmd != None {
			target.TestCommand, target.TestCommands = decodeCommands(s, args[2])
		}
		target.TestTimeout = sizeAndTimeout(s, size, args[25], s.state.Config.Test.Timeout)
		target.TestSandbox = isTruthy(21)
		target.NoTestOutput = isTruthy(22)
	}
	return target
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
		for _, url := range args[38].(pyList) {
			t.AddSource(core.URLLabel(url.(pyString)))
		}
	} else {
		addMaybeNamed(s, "srcs", args[3], t.AddSource, t.AddNamedSource, false, false)
	}
	addMaybeNamed(s, "tools", args[9], t.AddTool, t.AddNamedTool, true, true)
	addMaybeNamed(s, "system_srcs", args[32], t.AddSource, nil, true, false)
	addMaybeNamed(s, "data", args[4], t.AddDatum, nil, false, false)
	addMaybeNamedOutput(s, "outs", args[5], t.AddOutput, t.AddNamedOutput, t, false)
	addMaybeNamedOutput(s, "optional_outs", args[35], t.AddOptionalOutput, nil, t, true)
	addMaybeNamedOutput(s, "test_outputs", args[31], t.AddTestOutput, nil, t, false)
	addDependencies(s, "deps", args[6], t, false, false)
	addDependencies(s, "exported_deps", args[7], t, true, false)
	addDependencies(s, "internal_deps", args[39], t, false, true)
	addStrings(s, "labels", args[10], t.AddLabel)
	addStrings(s, "hashes", args[12], t.AddHash)
	addStrings(s, "licences", args[30], t.AddLicence)
	addStrings(s, "requires", args[28], t.AddRequire)
	addStrings(s, "visibility", args[11], func(str string) {
		t.Visibility = append(t.Visibility, parseVisibility(s, str))
	})
	addMaybeNamedSecret(s, "secrets", args[8], t.AddSecret, t.AddNamedSecret, t, true)
	addProvides(s, "provides", args[29], t)
	if f := callbackFunction(s, "pre_build", args[26], 1, "argument"); f != nil {
		t.PreBuildFunction = &preBuildFunction{f: f, s: s}
	}
	if f := callbackFunction(s, "post_build", args[27], 2, "arguments"); f != nil {
		t.PostBuildFunction = &postBuildFunction{f: f, s: s}
	}
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
				l, ok := asList(v)
				s.Assert(ok, "Values of %s must be lists of strings", name)
				for _, li := range l {
					if bi := parseBuildInput(s, li, name, systemAllowed, tool); bi != nil {
						named(k, bi)
					}
				}
			}
		}
	} else if obj != None {
		s.Assert(false, "Argument %s must be a list or dict, not %s", name, obj.Type())
	}
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
				checkSubDir(s, out.String())
				anon(string(out))
				if !optional || !strings.HasPrefix(string(out), "*") {
					s.pkg.MustRegisterOutput(string(out), t)
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
					checkSubDir(s, out.String())
					named(k, string(out))
					if !optional || !strings.HasPrefix(string(out), "*") {
						s.pkg.MustRegisterOutput(string(out), t)
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
		s.Assert(strings.HasPrefix(secret, "/") || strings.HasPrefix(secret, "~"),
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
		target.AddMaybeExportedDependency(checkLabel(s, core.ParseBuildLabelContext(str, s.pkg)), exported, false, internal)
	})
}

// addStrings adds an arbitrary set of strings to the target (e.g. labels etc).
func addStrings(s *scope, name string, obj pyObject, f func(string)) {
	if obj != nil && obj != None {
		l, ok := asList(obj)
		s.Assert(ok, "Argument %s must be a list, not %s", name, obj.Type())
		for _, li := range l {
			str, ok := li.(pyString)
			s.Assert(ok || li == None, "%s must be strings", name)
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
		s.Assert(ok, "Argument %s must be a dict, not %s", name, obj.Type())
		for k, v := range d {
			str, ok := v.(pyString)
			s.Assert(ok, "%s values must be strings", name)
			t.AddProvide(k, checkLabel(s, core.ParseBuildLabelContext(string(str), s.pkg)))
		}
	}
}

// parseVisibility converts a visibility string to a build label.
// Mostly they are just build labels but other things are allowed too (e.g. "PUBLIC").
func parseVisibility(s *scope, vis string) core.BuildLabel {
	if vis == "PUBLIC" || (s.state.Config.Bazel.Compatibility && vis == "//visibility:public") {
		return core.WholeGraph[0]
	}
	l := core.ParseBuildLabelContext(vis, s.pkg)
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
		if tool && s.pkg.Subrepo != nil && s.pkg.Subrepo.IsCrossCompile {
			// Tools always use the host configuration.
			// TODO(peterebden): this should really use something involving named output labels;
			//                   right now we don't have a package handy to call that but we
			//                   don't use them for tools anywhere either...
			return checkLabel(s, core.ParseBuildLabel(src, s.pkg.Name))
		}
		label := core.MustParseNamedOutputLabel(src, s.pkg)
		if l := label.Label(); l != nil {
			checkLabel(s, *l)
		}
		return label
	}
	s.Assert(src != "", "Empty source path")
	s.Assert(!strings.Contains(src, "../"), "%s is an invalid path; build target paths can't contain ../", src)
	checkSubDir(s, src)
	if src[0] == '/' || src[0] == '~' {
		s.Assert(systemAllowed, "%s is an absolute path; that's not allowed", src)
		return core.SystemFileLabel{Path: src}
	} else if tool {
		// "go" as a source is interpreted as a file, as a tool it's interpreted as something on the PATH.
		return core.SystemPathLabel{Name: src, Path: s.state.Config.Path()}
	}
	// Make sure it's not the actual build file.
	for _, filename := range s.state.Config.Parse.BuildFileName {
		s.Assert(filename != src, "You can't specify the BUILD file as an input to a rule")
	}
	return core.NewFileLabel(src, s.pkg)
}

// checkLabel checks that the given build label is not a pseudo-label.
// These are disallowed in (nearly) all contexts.
func checkLabel(s *scope, label core.BuildLabel) core.BuildLabel {
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
	s := f.f.scope.NewPackagedScope(f.f.scope.state.Graph.PackageOrDie(target.Label))
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
	s := f.f.scope.NewPackagedScope(f.f.scope.state.Graph.PackageOrDie(target.Label))
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

// asDict converts an object to a pyDict, accounting for frozen dicts.
func asDict(obj pyObject) (pyDict, bool) {
	if d, ok := obj.(pyDict); ok {
		return d, true
	} else if d, ok := obj.(pyFrozenDict); ok {
		return d.pyDict, true
	}
	return nil, false
}

// Target is in a subdirectory, check nobody else owns that.
func checkSubDir(s *scope, src string) {
	if strings.Contains(src, "/") {
		// Target is in a subdirectory, check nobody else owns that.
		sr := s.pkg.SourceRoot()
		for dir := path.Dir(path.Join(sr, src)); dir != sr && dir != "." && dir != "/"; dir = path.Dir(dir) {
			s.Assert(!fs.IsPackage(s.state.Config.Parse.BuildFileName, dir), "Trying to use file %s, but that belongs to another package (%s)", src, dir)
		}
	}
}
