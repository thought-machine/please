package asp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/manifoldco/promptui"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// A nativeFunc is a function that implements a builtin function natively.
type nativeFunc func(*scope, []pyObject) pyObject

// registerBuiltins sets up the "special" builtins that map to native code.
func registerBuiltins(s *scope) {
	const varargs = true
	const kwargs = true
	setNativeCode(s, "build_rule", buildRule)
	setNativeCode(s, "tag", tag)
	setNativeCode(s, "subrepo", subrepo)
	setNativeCode(s, "fail", builtinFail)
	setNativeCode(s, "subinclude", subinclude, varargs)
	setNativeCode(s, "load", bazelLoad, varargs)
	setNativeCode(s, "package", pkg, false, kwargs)
	setNativeCode(s, "sorted", sorted)
	setNativeCode(s, "reversed", reversed)
	setNativeCode(s, "isinstance", isinstance)
	setNativeCode(s, "range", pyRange)
	setNativeCode(s, "enumerate", enumerate)
	setNativeCode(s, "zip", zip, varargs)
	setNativeCode(s, "any", anyFunc)
	setNativeCode(s, "all", allFunc)
	setNativeCode(s, "min", min)
	setNativeCode(s, "max", max)
	setNativeCode(s, "len", lenFunc)
	setNativeCode(s, "glob", glob)
	setNativeCode(s, "bool", boolType)
	setNativeCode(s, "int", intType)
	setNativeCode(s, "str", strType)
	setNativeCode(s, "join_path", joinPath, varargs)
	setNativeCode(s, "get_base_path", packageName)
	setNativeCode(s, "package_name", packageName)
	setNativeCode(s, "subrepo_name", subrepoName)
	setNativeCode(s, "canonicalise", canonicalise)
	setNativeCode(s, "get_labels", getLabels)
	setNativeCode(s, "add_label", addLabel)
	setNativeCode(s, "add_dep", addDep)
	setNativeCode(s, "add_data", addData)
	setNativeCode(s, "add_out", addOut)
	setNativeCode(s, "get_outs", getOuts)
	setNativeCode(s, "get_named_outs", getNamedOuts)
	setNativeCode(s, "add_licence", addLicence)
	setNativeCode(s, "get_licences", getLicences)
	setNativeCode(s, "get_command", getCommand)
	setNativeCode(s, "set_command", setCommand)
	setNativeCode(s, "json", valueAsJSON)
	setNativeCode(s, "breakpoint", breakpoint)
	setNativeCode(s, "is_semver", isSemver)
	setNativeCode(s, "semver_check", semverCheck)
	setNativeCode(s, "looks_like_build_label", looksLikeBuildLabel)
	s.interpreter.stringMethods = map[string]*pyFunc{
		"join":         setNativeCode(s, "join", strJoin),
		"split":        setNativeCode(s, "split", strSplit),
		"replace":      setNativeCode(s, "replace", strReplace),
		"partition":    setNativeCode(s, "partition", strPartition),
		"rpartition":   setNativeCode(s, "rpartition", strRPartition),
		"startswith":   setNativeCode(s, "startswith", strStartsWith),
		"endswith":     setNativeCode(s, "endswith", strEndsWith),
		"lstrip":       setNativeCode(s, "lstrip", strLStrip),
		"rstrip":       setNativeCode(s, "rstrip", strRStrip),
		"removeprefix": setNativeCode(s, "removeprefix", strRemovePrefix),
		"removesuffix": setNativeCode(s, "removesuffix", strRemoveSuffix),
		"strip":        setNativeCode(s, "strip", strStrip),
		"find":         setNativeCode(s, "find", strFind),
		"rfind":        setNativeCode(s, "rfind", strRFind),
		"format":       setNativeCode(s, "format", strFormat),
		"count":        setNativeCode(s, "count", strCount),
		"upper":        setNativeCode(s, "upper", strUpper),
		"lower":        setNativeCode(s, "lower", strLower),
	}
	s.interpreter.stringMethods["format"].kwargs = true
	s.interpreter.dictMethods = map[string]*pyFunc{
		"get":        setNativeCode(s, "get", dictGet),
		"setdefault": s.Lookup("setdefault").(*pyFunc),
		"keys":       setNativeCode(s, "keys", dictKeys),
		"items":      setNativeCode(s, "items", dictItems),
		"values":     setNativeCode(s, "values", dictValues),
		"copy":       setNativeCode(s, "copy", dictCopy),
	}
	s.interpreter.configMethods = map[string]*pyFunc{
		"get":        setNativeCode(s, "config_get", configGet),
		"setdefault": s.Lookup("setdefault").(*pyFunc),
	}
	if s.state.Config.Parse.GitFunctions {
		setNativeCode(s, "git_branch", execGitBranch)
		setNativeCode(s, "git_commit", execGitCommit)
		setNativeCode(s, "git_show", execGitShow)
		setNativeCode(s, "git_state", execGitState)
	}
	setLogCode(s, "debug", log.Debug)
	setLogCode(s, "info", log.Info)
	setLogCode(s, "notice", log.Notice)
	setLogCode(s, "warning", log.Warning)
	setLogCode(s, "error", log.Errorf)
	setLogCode(s, "fatal", log.Fatalf)
}

// registerSubincludePackage sets up the package for remote subincludes.
func registerSubincludePackage(s *scope) {
	// Another small hack - replace the code for these two with native code, must be done after the
	// declarations which are in misc_rules.
	buildRule := s.Lookup("build_rule").(*pyFunc)
	f := setNativeCode(s, "filegroup", filegroup)
	f.args = buildRule.args
	f.argIndices = buildRule.argIndices
	f.defaults = buildRule.defaults
	f.constants = buildRule.constants
	f.types = buildRule.types
	f.args = buildRule.args
	f.argIndices = buildRule.argIndices
	f.defaults = buildRule.defaults
	f.constants = buildRule.constants
	f.types = buildRule.types
}

func setNativeCode(s *scope, name string, code nativeFunc, flags ...bool) *pyFunc {
	f := s.Lookup(name).(*pyFunc)
	f.nativeCode = code
	f.code = nil // Might as well save a little memory here
	if len(flags) != 0 {
		f.varargs = flags[0]
		f.kwargs = len(flags) > 1 && flags[1]
	} else {
		f.argPool = &sync.Pool{
			New: func() interface{} {
				return make([]pyObject, len(f.args))
			},
		}
	}
	return f
}

// setLogCode specialises setNativeCode for handling the log functions (of which there are a few)
func setLogCode(s *scope, name string, f func(format string, args ...interface{})) {
	setNativeCode(s, name, func(s *scope, args []pyObject) pyObject {
		if str, ok := args[0].(pyString); ok {
			l := make([]interface{}, len(args))
			for i, arg := range args {
				l[i] = arg
			}
			f("//%s: %s", s.pkgFilename(), fmt.Sprintf(string(str), l[1:]...))
			return None
		}
		f("//%s: %s", s.pkgFilename(), args)
		return None
	}).varargs = true
}

// buildRule implements the build_rule() builtin function.
// This is the main interface point; every build rule ultimately calls this to add
// new objects to the build graph.
func buildRule(s *scope, args []pyObject) pyObject {
	s.NAssert(s.pkg == nil, "Cannot create new build rules in this scope")
	// We need to set various defaults from config here; it is useful to put it on the rule but not often so
	// because most rules pass them through anyway.
	// TODO(peterebden): when we get rid of the old parser, put these defaults on all the build rules and
	//                   get rid of this.
	args[visibilityBuildRuleArgIdx] = defaultFromConfig(s.config, args[visibilityBuildRuleArgIdx], "DEFAULT_VISIBILITY")
	args[testOnlyBuildRuleArgIdx] = defaultFromConfig(s.config, args[testOnlyBuildRuleArgIdx], "DEFAULT_TESTONLY")
	args[licencesBuildRuleArgIdx] = defaultFromConfig(s.config, args[licencesBuildRuleArgIdx], "DEFAULT_LICENCES")
	args[sandboxBuildRuleArgIdx] = defaultFromConfig(s.config, args[sandboxBuildRuleArgIdx], "BUILD_SANDBOX")
	args[testSandboxBuildRuleArgIdx] = defaultFromConfig(s.config, args[testSandboxBuildRuleArgIdx], "TEST_SANDBOX")

	// Don't want to remote execute a target if we need system sources
	if args[systemSrcsBuildRuleArgIdx] != None {
		args[localBuildRuleArgIdx] = pyString("True")
	}

	target := createTarget(s, args)
	s.Assert(s.pkg.Target(target.Label.Name) == nil, "Duplicate build target in %s: %s", s.pkg.Name, target.Label.Name)
	populateTarget(s, target, args)
	s.state.AddTarget(s.pkg, target)
	if s.Callback {
		target.AddedPostBuild = true
	}

	if s.parsingFor != nil && s.parsingFor.label == target.Label {
		if err := s.state.ActivateTarget(s.pkg, s.parsingFor.label, s.parsingFor.dependent, s.mode); err != nil {
			s.Error("%v", err)
		}
	}

	return pyString(":" + target.Label.Name)
}

// defaultFromConfig sets a default value from the config if the property isn't set.
func defaultFromConfig(config *pyConfig, arg pyObject, name string) pyObject {
	if arg == nil || arg == None {
		return config.Get(name, arg)
	}
	return arg
}

// filegroup implements the filegroup() builtin.
func filegroup(s *scope, args []pyObject) pyObject {
	args[1] = filegroupCommand
	return buildRule(s, args)
}

// pkg implements the package() builtin function.
func pkg(s *scope, args []pyObject) pyObject {
	s.Assert(s.pkg.NumTargets() == 0, "package() must be called before any build targets are defined")
	for k, v := range s.locals {
		k = strings.ToUpper(k)
		configVal := s.config.Get(k, nil)
		s.Assert(configVal != nil, "error calling package(): %s is not a known config value", k)

		// Merge in the existing config for dictionaries
		if overrides, ok := v.(pyDict); ok {
			if pluginConfig, ok := configVal.(pyDict); ok {
				newPluginConfig := pluginConfig.Copy()
				for pluginKey, override := range overrides {
					pluginKey = strings.ToUpper(pluginKey)
					if _, ok := newPluginConfig[pluginKey]; !ok {
						s.Error("error calling package(): %s.%s is not a known config value", k, pluginKey)
					}

					newPluginConfig.IndexAssign(pyString(pluginKey), override)
				}
				v = newPluginConfig
			} else {
				s.Error("error calling package(): can't assign a dict to %s as it's not a dict", k)
			}
		}
		s.config.IndexAssign(pyString(k), v)
	}
	return None
}

func tag(s *scope, args []pyObject) pyObject {
	name := args[0].String()
	tag := args[1].String()

	return pyString(tagName(name, tag))
}

// tagName applies the given tag to a target name.
func tagName(name, tag string) string {
	if name[0] != '_' {
		name = "_" + name
	}
	if strings.ContainsRune(name, '#') {
		name += "_"
	} else {
		name += "#"
	}
	return name + tag
}

// bazelLoad implements the load() builtin, which is only available for Bazel compatibility.
func bazelLoad(s *scope, args []pyObject) pyObject {
	s.Assert(s.state.Config.Bazel.Compatibility, "load() is only available in Bazel compatibility mode. See `plz help bazel` for more information.")
	// The argument always looks like a build label, but it is not really one (i.e. there is no BUILD file that defines it).
	// We do not support their legacy syntax here (i.e. "/tools/build_rules/build_test" etc).
	l := s.parseLabelInContextPkg(string(args[0].(pyString)))
	filename := filepath.Join(l.PackageName, l.Name)
	if l.Subrepo != "" {
		subrepo := s.state.Graph.Subrepo(l.Subrepo)
		if subrepo == nil || (subrepo.Target != nil && subrepo != s.contextPackage().Subrepo) {
			subincludeTarget(s, l)
			subrepo = s.state.Graph.SubrepoOrDie(l.Subrepo)
		}
		filename = subrepo.Dir(filename)
	}
	s.SetAll(s.interpreter.Subinclude(s, filename, l, false), false)
	return None
}

// WaitForSubincludedTarget drops the interpreter lock and waits for the subincluded target to be built. This is
// important to keep us from deadlocking all available parser threads (easy to happen if they're all waiting on a
// single target which now can't start)
func (s *scope) WaitForSubincludedTarget(l, dependent core.BuildLabel) *core.BuildTarget {
	s.interpreter.limiter.Release()
	defer s.interpreter.limiter.Acquire()

	return s.state.WaitForTargetAndEnsureDownload(l, dependent, s.mode.IsPreload())
}

// builtinFail raises an immediate error that can't be intercepted.
func builtinFail(s *scope, args []pyObject) pyObject {
	s.Error(string(args[0].(pyString)))
	return None
}

func subinclude(s *scope, args []pyObject) pyObject {
	if s.contextPackage() == nil {
		s.Error("cannot subinclude from this scope")
	}
	for _, arg := range args {
		t := subincludeTarget(s, s.parseLabelInContextPkg(string(arg.(pyString))))
		s.Assert(s.contextPackage().Label().CanSee(s.state, t), "Target %s isn't visible to be subincluded into %s", t.Label, s.contextPackage().Label())

		incPkgState := s.state
		if t.Label.Subrepo != "" {
			subrepo := s.state.Graph.SubrepoOrDie(t.Label.Subrepo)
			incPkgState = subrepo.State
		}
		s.interpreter.loadPluginConfig(s, incPkgState)

		for _, out := range t.Outputs() {
			s.SetAll(s.interpreter.Subinclude(s, filepath.Join(t.OutDir(), out), t.Label, false), false)
		}
	}
	return None
}

// subincludeTarget returns the target for a subinclude() call to a label.
// It blocks until the target exists and is built.
func subincludeTarget(s *scope, l core.BuildLabel) *core.BuildTarget {
	s.NAssert(l.IsPseudoTarget(), "Can't pass :all or /... to subinclude()")

	pkg := s.contextPackage()
	pkgLabel := pkg.Label()

	// If we're including from a subrepo, or if we're in a subrepo and including from a different subrepo, make sure
	// that package is parsed to avoid locking. Locks can occur when the target's package also subincludes that target.
	//
	// When this happens, both parse thread "WaitForBuiltTarget" expecting the other to queue the target to be built.
	//
	// By parsing the package first, the subrepo package's subinclude will queue the subrepo target to be built before
	// we call WaitForSubincludedTarget below avoiding the lockup.
	subrepoLabel := l.SubrepoLabel(s.state, "")
	if l.Subrepo != "" && subrepoLabel.PackageName != pkg.Name && l.Subrepo != pkg.SubrepoName {
		subrepoPackageLabel := core.BuildLabel{
			PackageName: subrepoLabel.PackageName,
			Subrepo:     subrepoLabel.Subrepo,
			Name:        "all",
		}
		s.state.WaitForPackage(subrepoPackageLabel, pkgLabel, s.mode|core.ParseModeForSubinclude)
	}

	// isLocal is true when this subinclude target in the current package being parsed
	isLocal := l.Subrepo == pkgLabel.Subrepo && l.PackageName == pkgLabel.PackageName

	// If the subinclude is local to this package, it must already exist in the graph. If it already exists in the graph
	// but isn't activated, we should activate it otherwise WaitForSubincludedTarget might block. This can happen when
	// another package also subincludes this target, and queues it first.
	if isLocal && s.pkg != nil {
		t := s.state.Graph.Target(l)
		if t == nil {
			s.Error("Target :%s is not defined in this package; it has to be defined before the subinclude() call", l.Name)
		}
		if t.State() < core.Active {
			if err := s.state.ActivateTarget(s.pkg, l, pkgLabel, s.mode|core.ParseModeForSubinclude); err != nil {
				s.Error("Failed to activate subinclude target: %v", err)
			}
		}
	}

	t := s.WaitForSubincludedTarget(l, pkgLabel)

	// TODO(jpoole): when pkg is nil, that means this subinclude was made by another subinclude. We're currently loosing
	// this information here. We probably need a way to transitively record the subincludes.
	if s.pkg != nil {
		s.pkg.RegisterSubinclude(l)
	}
	return t
}

func lenFunc(s *scope, args []pyObject) pyObject {
	return objLen(args[0])
}

func objLen(obj pyObject) pyInt {
	switch t := obj.(type) {
	case pyList:
		return pyInt(len(t))
	case pyFrozenList:
		return pyInt(len(t.pyList))
	case pyDict:
		return pyInt(len(t))
	case pyFrozenDict:
		return pyInt(len(t.pyDict))
	case pyString:
		return pyInt(len(t))
	}
	panic("object of type " + obj.Type() + " has no len()")
}

func isinstance(s *scope, args []pyObject) pyObject {
	obj := args[0]
	typesArg := args[1]

	var types pyList

	if l, ok := typesArg.(pyList); ok {
		types = l
	} else {
		types = pyList{typesArg}
	}

	for _, li := range types {
		// Special case for 'str' and so forth that are functions but also types.
		if lif, ok := li.(*pyFunc); ok && isType(obj, lif.name) {
			return True
		} else if _, ok := obj.(*pyFunc); ok {
			continue // reflect would always return true
		} else if reflect.TypeOf(obj) == reflect.TypeOf(li) {
			return True
		}
	}
	if _, ok := obj.(*pyFunc); ok {
		return False // reflect would always return true
	}
	return newPyBool(reflect.TypeOf(obj) == reflect.TypeOf(typesArg))
}

func isType(obj pyObject, name string) bool {
	switch obj.(type) {
	case pyBool:
		return name == "bool" || name == "int" // N.B. For compatibility with old assert statements
	case pyInt:
		return name == "int"
	case pyString:
		return name == "str"
	case pyList:
		return name == "list"
	case pyDict:
		return name == "dict"
	case *pyConfig:
		return name == "config"
	case *pyFunc:
		return name == "callable"
	}
	return false
}

func strJoin(s *scope, args []pyObject) pyObject {
	self := string(args[0].(pyString))
	seq := asStringList(s, args[1], "seq")
	return pyString(strings.Join(seq, self))
}

func strSplit(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	on := args[1].(pyString)
	return fromStringList(strings.Split(string(self), string(on)))
}

func strReplace(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	old := args[1].(pyString)
	new := args[2].(pyString)
	return pyString(strings.ReplaceAll(string(self), string(old), string(new)))
}

func strPartition(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	sep := args[1].(pyString)
	if idx := strings.Index(string(self), string(sep)); idx != -1 {
		return pyList{self[:idx], self[idx : idx+len(sep)], self[idx+len(sep):]}
	}
	return pyList{self, pyString(""), pyString("")}
}

func strRPartition(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	sep := args[1].(pyString)
	if idx := strings.LastIndex(string(self), string(sep)); idx != -1 {
		return pyList{self[:idx], self[idx : idx+len(sep)], self[idx+len(sep):]}
	}
	return pyList{pyString(""), pyString(""), self}
}

func strStartsWith(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	x := args[1].(pyString)
	return newPyBool(strings.HasPrefix(string(self), string(x)))
}

func strEndsWith(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	x := args[1].(pyString)
	return newPyBool(strings.HasSuffix(string(self), string(x)))
}

func strLStrip(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	cutset := args[1].(pyString)
	return pyString(strings.TrimLeft(string(self), string(cutset)))
}

func strRStrip(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	cutset := args[1].(pyString)
	return pyString(strings.TrimRight(string(self), string(cutset)))
}

func strStrip(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	cutset := args[1].(pyString)
	return pyString(strings.Trim(string(self), string(cutset)))
}

func strRemovePrefix(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	prefix := args[1].(pyString)
	return pyString(strings.TrimPrefix(string(self), string(prefix)))
}

func strRemoveSuffix(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	suffix := args[1].(pyString)
	return pyString(strings.TrimSuffix(string(self), string(suffix)))
}

func strFind(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	needle := args[1].(pyString)
	return pyInt(strings.Index(string(self), string(needle)))
}

func strRFind(s *scope, args []pyObject) pyObject {
	self := args[0].(pyString)
	needle := args[1].(pyString)
	return pyInt(strings.LastIndex(string(self), string(needle)))
}

func strFormat(s *scope, args []pyObject) pyObject {
	self := string(args[0].(pyString))
	for k, v := range s.locals {
		self = strings.ReplaceAll(self, "{"+k+"}", v.String())
	}
	for _, arg := range args[1:] {
		self = strings.Replace(self, "{}", arg.String(), 1)
	}
	return pyString(strings.ReplaceAll(strings.ReplaceAll(self, "{{", "{"), "}}", "}"))
}

func strCount(s *scope, args []pyObject) pyObject {
	self := string(args[0].(pyString))
	needle := string(args[1].(pyString))
	return pyInt(strings.Count(self, needle))
}

func strUpper(s *scope, args []pyObject) pyObject {
	self := string(args[0].(pyString))
	return pyString(strings.ToUpper(self))
}

func strLower(s *scope, args []pyObject) pyObject {
	self := string(args[0].(pyString))
	return pyString(strings.ToLower(self))
}

func boolType(s *scope, args []pyObject) pyObject {
	return newPyBool(args[0].IsTruthy())
}

func intType(s *scope, args []pyObject) pyObject {
	i, err := strconv.Atoi(string(args[0].(pyString)))
	s.Assert(err == nil, "%s", err)
	return pyInt(i)
}

func strType(s *scope, args []pyObject) pyObject {
	return pyString(args[0].String())
}

func glob(s *scope, args []pyObject) pyObject {
	include := pyStrOrListAsList(s, args[0], "include")
	exclude := pyStrOrListAsList(s, args[1], "exclude")
	hidden := args[2].IsTruthy()
	includeSymlinks := args[3].IsTruthy()
	allowEmpty := args[4].IsTruthy()
	exclude = append(exclude, s.state.Config.Parse.BuildFileName...)
	if s.globber == nil {
		s.globber = fs.NewGlobber(s.state.Config.Parse.BuildFileName)
	}

	glob := s.globber.Glob(s.pkg.SourceRoot(), include, exclude, hidden, includeSymlinks)
	if !allowEmpty && len(glob) == 0 {
		// Strip build file name from exclude list for error message
		exclude = exclude[:len(exclude)-len(s.state.Config.Parse.BuildFileName)]
		log.Fatalf("glob(include=%s, exclude=%s) in %s returned no files. If this is intended, set allow_empty=True on the glob.", include, exclude, s.pkg.Filename)
	}

	return fromStringList(glob)
}

func pyStrOrListAsList(s *scope, arg pyObject, name string) []string {
	if str, ok := arg.(pyString); ok {
		return []string{str.String()}
	}
	return asStringList(s, arg, name)
}

func asStringList(s *scope, arg pyObject, name string) []string {
	if fl, ok := arg.(pyFrozenList); ok {
		arg = fl.pyList
	}
	l, ok := arg.(pyList)
	s.Assert(ok, "argument %s must be a list", name)
	sl := make([]string, len(l))
	for i, x := range l {
		sx, ok := x.(pyString)
		s.Assert(ok, "%s must be a list of strings", name)
		sl[i] = string(sx)
	}
	return sl
}

func fromStringList(l []string) pyList {
	ret := make(pyList, len(l))
	for i, s := range l {
		ret[i] = pyString(s)
	}
	return ret
}

func configGet(s *scope, args []pyObject) pyObject {
	self := args[0].(*pyConfig)
	return self.Get(string(args[1].(pyString)), args[2])
}

func dictGet(s *scope, args []pyObject) pyObject {
	self := args[0].(pyDict)
	sk, ok := args[1].(pyString)
	s.Assert(ok, "dict keys must be strings, not %s", args[1].Type())
	if ret, present := self[string(sk)]; present {
		return ret
	}
	return args[2]
}

func dictKeys(s *scope, args []pyObject) pyObject {
	self := args[0].(pyDict)
	ret := make(pyList, len(self))
	for i, k := range self.Keys() {
		ret[i] = pyString(k)
	}
	return ret
}

func dictValues(s *scope, args []pyObject) pyObject {
	self := args[0].(pyDict)
	ret := make(pyList, len(self))
	for i, k := range self.Keys() {
		ret[i] = self[k]
	}
	return ret
}

func dictItems(s *scope, args []pyObject) pyObject {
	self := args[0].(pyDict)
	ret := make(pyList, len(self))
	for i, k := range self.Keys() {
		ret[i] = pyList{pyString(k), self[k]}
	}
	return ret
}

func dictCopy(s *scope, args []pyObject) pyObject {
	self := args[0].(pyDict)
	ret := make(pyDict, len(self))
	for k, v := range self {
		ret[k] = v
	}
	return ret
}

func sorted(s *scope, args []pyObject) pyObject {
	l, ok := args[0].(pyList)
	s.Assert(ok, "unsortable type %s", args[0].Type())
	l = l[:]
	sort.Slice(l, func(i, j int) bool { return l[i].Operator(LessThan, l[j]).IsTruthy() })
	return l
}

func reversed(s *scope, args []pyObject) pyObject {
	l, ok := args[0].(pyList)
	s.Assert(ok, "irreversible type %s", args[0].Type())
	l = l[:]
	// TODO(chrisnovakovic): replace with slices.Reverse after upgrading to Go 1.21
	for i, j := 0, len(l)-1; i < j; i, j = i+1, j-1 {
		l[i], l[j] = l[j], l[i]
	}
	return l
}

func joinPath(s *scope, args []pyObject) pyObject {
	l := make([]string, len(args))
	for i, arg := range args {
		l[i] = string(arg.(pyString))
	}
	return pyString(filepath.Join(l...))
}

func looksLikeBuildLabel(s *scope, args []pyObject) pyObject {
	return pyBool(core.LooksLikeABuildLabel(args[0].String()))
}

// scopeOrSubincludePackage is like (*scope).contextPackage() package but allows the option to force the use the
// subinclude package
func scopeOrSubincludePackage(s *scope, subinclude bool) (*core.Package, error) {
	if subinclude {
		pkg := s.subincludePackage()
		if pkg == nil {
			return nil, errors.New("not in a subinclude scope")
		}
		return pkg, nil
	}
	return s.contextPackage(), nil
}

func packageName(s *scope, args []pyObject) pyObject {
	const (
		labelArgIdx = iota
		contextArgIdx
	)

	pkg, err := scopeOrSubincludePackage(s, args[contextArgIdx].IsTruthy())
	if err != nil {
		s.Error("cannot call package_name() from this scope: %v", err)
	}

	if args[labelArgIdx].IsTruthy() {
		return pyString(s.parseLabelInPackage(string(args[labelArgIdx].(pyString)), pkg).PackageName)
	}

	return pyString(pkg.Name)
}

func subrepoName(s *scope, args []pyObject) pyObject {
	const (
		labelArgIdx = iota
		contextArgIdx
	)

	pkg, err := scopeOrSubincludePackage(s, args[contextArgIdx].IsTruthy())
	if err != nil {
		s.Error("cannot call subrepo_name() from this scope: %v", err)
	}

	if args[labelArgIdx].IsTruthy() {
		l := s.parseAnnotatedLabelInPackage(string(args[labelArgIdx].(pyString)), pkg)
		if label, _ := l.Label(); label.Subrepo != "" {
			return pyString(label.Subrepo)
		}
	}

	return pyString(pkg.SubrepoName)
}

func canonicalise(s *scope, args []pyObject) pyObject {
	const (
		labelArgIdx = iota
		contextArgIdx
	)
	pkg, err := scopeOrSubincludePackage(s, args[contextArgIdx].IsTruthy())
	if err != nil {
		s.Error("Cannot call canonicalise() from this scope: %v", err)
	}
	label := s.parseLabelInPackage(string(args[labelArgIdx].(pyString)), pkg)
	return pyString(label.String())
}

func pyRange(s *scope, args []pyObject) pyObject {
	start := args[0].(pyInt)
	stop, isInt := args[1].(pyInt)
	step := args[2].(pyInt)
	if !isInt {
		// Stop not passed so we start at 0 and start is the stop.
		stop = start
		start = 0
	}
	ret := make(pyList, 0, stop-start)
	for i := start; i < stop; i += step {
		ret = append(ret, i)
	}
	return ret
}

func enumerate(s *scope, args []pyObject) pyObject {
	l, ok := args[0].(pyList)
	s.Assert(ok, "Argument to enumerate must be a list, not %s", args[0].Type())
	ret := make(pyList, len(l))
	for i, li := range l {
		ret[i] = pyList{pyInt(i), li}
	}
	return ret
}

func anyFunc(s *scope, args []pyObject) pyObject {
	l, ok := args[0].(pyList)
	s.Assert(ok, "Argument to any must be a list, not %s", args[0].Type())
	for _, li := range l {
		if li.IsTruthy() {
			return True
		}
	}
	return False
}

func allFunc(s *scope, args []pyObject) pyObject {
	l, ok := args[0].(pyList)
	s.Assert(ok, "Argument to all must be a list, not %s", args[0].Type())
	for _, li := range l {
		if !li.IsTruthy() {
			return False
		}
	}
	return True
}

func min(s *scope, args []pyObject) pyObject {
	return extreme(s, args, LessThan)
}

func max(s *scope, args []pyObject) pyObject {
	return extreme(s, args, GreaterThan)
}

func extreme(s *scope, args []pyObject, cmp Operator) pyObject {
	l, isList := args[0].(pyList)
	key, isFunc := args[1].(*pyFunc)
	s.Assert(isList, "Argument seq must be a list, not %s", args[0].Type())
	s.Assert(len(l) > 0, "Argument seq must contain at least one item")
	if key != nil {
		s.Assert(isFunc, "Argument key must be callable, not %s", args[1].Type())
	}
	var cret, ret pyObject
	for i, li := range l {
		cli := li
		if key != nil {
			c := &Call{
				Arguments: []CallArgument{{
					Value: Expression{Optimised: &OptimisedExpression{Constant: li}},
				}},
			}
			cli = key.Call(s, c)
		}
		if i == 0 || cli.Operator(cmp, cret).IsTruthy() {
			cret = cli
			ret = li
		}
	}
	return ret
}

func zip(s *scope, args []pyObject) pyObject {
	lastLen := 0
	for i, seq := range args {
		si, ok := seq.(pyList)
		s.Assert(ok, "Arguments to zip must be lists, not %s", si.Type())
		// This isn't a restriction in Python but I can't be bothered handling all the stuff that real zip does.
		s.Assert(i == 0 || lastLen == len(si), "All arguments to zip must have the same length")
		lastLen = len(si)
	}
	ret := make(pyList, lastLen)
	for i := range ret {
		r := make(pyList, len(args))
		for j, li := range args {
			r[j] = li.(pyList)[i]
		}
		ret[i] = r
	}
	return ret
}

// getLabels returns the set of labels for a build target and its transitive dependencies.
// The labels are filtered by the given prefix, which is stripped from the returned labels.
// Two formats are supported here: either passing just the name of a target in the current
// package, or a build label referring specifically to one.
func getLabels(s *scope, args []pyObject) pyObject {
	name := string(args[0].(pyString))
	prefix := string(args[1].(pyString))
	all := args[2].IsTruthy()
	transitive := args[3].IsTruthy()
	if core.LooksLikeABuildLabel(name) {
		label := core.ParseBuildLabel(name, s.pkg.Name)
		return getLabelsInternal(s.state.Graph.TargetOrDie(label), prefix, core.Built, all, transitive)
	}
	target := getTargetPost(s, name)
	return getLabelsInternal(target, prefix, core.Building, all, transitive)
}

// addLabel adds a set of labels to the named rule
func addLabel(s *scope, args []pyObject) pyObject {
	name := string(args[0].(pyString))

	var target *core.BuildTarget
	if core.LooksLikeABuildLabel(name) {
		label := core.ParseBuildLabel(name, s.pkg.Name)
		target = s.state.Graph.TargetOrDie(label)
	} else {
		target = getTargetPost(s, name)
	}

	target.AddLabel(args[1].String())

	return None
}

func getLabelsInternal(target *core.BuildTarget, prefix string, minState core.BuildTargetState, all, transitive bool) pyObject {
	if target.State() < minState {
		log.Fatalf("get_labels called on a target that is not yet built: %s", target.Label)
	}
	if all && !transitive {
		log.Fatalf("get_labels can't be called with all set to true when transitive is set to False")
	}
	labels := map[string]bool{}
	done := map[*core.BuildTarget]bool{}
	var getLabels func(*core.BuildTarget)
	getLabels = func(t *core.BuildTarget) {
		for _, label := range t.Labels {
			if strings.HasPrefix(label, prefix) {
				labels[strings.TrimSpace(strings.TrimPrefix(label, prefix))] = true
			}
		}
		if !transitive {
			return
		}
		done[t] = true
		if !t.OutputIsComplete || t == target || all {
			for _, dep := range t.Dependencies() {
				if !done[dep] {
					getLabels(dep)
				}
			}
		}
	}
	getLabels(target)
	ret := make([]string, len(labels))
	i := 0
	for label := range labels {
		ret[i] = label
		i++
	}
	sort.Strings(ret)
	return fromStringList(ret)
}

// getTargetPost is called by various functions to get a target from the current package.
// Panics if the target is not in the current package or has already been built.
func getTargetPost(s *scope, name string) *core.BuildTarget {
	target := s.pkg.Target(name)
	//nolint:staticcheck
	s.Assert(target != nil, "Unknown build target %s in %s", name, s.pkg.Name)
	// It'd be cheating to try to modify targets that're already built.
	// Prohibit this because it'd likely end up with nasty race conditions.
	s.Assert(target.State() < core.Built, "Attempted to modify target %s, but it's already built", target.Label) //nolint:staticcheck
	return target
}

// addDep adds a dependency to a target.
func addDep(s *scope, args []pyObject) pyObject {
	s.Assert(s.Callback, "can only be called from a pre- or post-build callback")
	target := getTargetPost(s, string(args[0].(pyString)))
	dep := s.parseLabelInPackage(string(args[1].(pyString)), s.pkg)
	exported := args[2].IsTruthy()
	target.AddMaybeExportedDependency(dep, exported, false, false)
	// Queue this dependency if it'll be needed.
	if target.State() > core.Inactive {
		err := s.state.QueueTarget(dep, target.Label, false, core.ParseModeNormal)
		s.Assert(err == nil, "%s", err)
	}
	return None
}

func addDatumToTargetAndMaybeQueue(s *scope, target *core.BuildTarget, datum core.BuildInput, systemAllowed, tool bool) {
	target.AddDatum(datum)
	// Queue this dependency if it'll be needed.
	if l, ok := datum.Label(); ok && target.State() > core.Inactive {
		err := s.state.QueueTarget(l, target.Label, false, core.ParseModeNormal)
		s.Assert(err == nil, "%s", err)
	}
}

func addNamedDatumToTargetAndMaybeQueue(s *scope, name string, target *core.BuildTarget, datum core.BuildInput, systemAllowed, tool bool) {
	target.AddNamedDatum(name, datum)
	// Queue this dependency if it'll be needed.
	if l, ok := datum.Label(); ok && target.State() > core.Inactive {
		err := s.state.QueueTarget(l, target.Label, false, core.ParseModeNormal)
		s.Assert(err == nil, "%s", err)
	}
}

// Add runtime dependencies to target
func addData(s *scope, args []pyObject) pyObject {
	s.Assert(s.Callback, "can only be called from a pre- or post-build callback")

	label := args[0]
	datum := args[1]
	target := getTargetPost(s, string(label.(pyString)))

	systemAllowed := false
	tool := false

	// add_data() builtin can take a string, list, or dict
	if isType(datum, "str") {
		if bi := parseBuildInput(s, datum, string(label.(pyString)), systemAllowed, tool); bi != nil {
			addDatumToTargetAndMaybeQueue(s, target, bi, systemAllowed, tool)
		}
	} else if isType(datum, "list") {
		for _, str := range datum.(pyList) {
			if bi := parseBuildInput(s, str, string(label.(pyString)), systemAllowed, tool); bi != nil {
				addDatumToTargetAndMaybeQueue(s, target, bi, systemAllowed, tool)
			}
		}
	} else if isType(datum, "dict") {
		for name, v := range datum.(pyDict) {
			for _, str := range v.(pyList) {
				if bi := parseBuildInput(s, str, string(label.(pyString)), systemAllowed, tool); bi != nil {
					addNamedDatumToTargetAndMaybeQueue(s, name, target, bi, systemAllowed, tool)
				}
			}
		}
	} else {
		log.Fatal("Unrecognised data type passed to add_data")
	}
	return None
}

// addOut adds an output to a target.
func addOut(s *scope, args []pyObject) pyObject {
	target := getTargetPost(s, string(args[0].(pyString)))
	name := string(args[1].(pyString))
	out := string(args[2].(pyString))
	if out == "" {
		target.AddOutput(name)
		s.pkg.MustRegisterOutput(s.state, name, target)
	} else {
		_, ok := target.EntryPoints[name]
		s.NAssert(ok, "Named outputs can't have the same name as entry points")

		target.AddNamedOutput(name, out)
		s.pkg.MustRegisterOutput(s.state, out, target)
	}
	return None
}

// getOuts gets the outputs of a target
func getOuts(s *scope, args []pyObject) pyObject {
	var target *core.BuildTarget
	if name := args[0].String(); core.LooksLikeABuildLabel(name) {
		label := core.ParseBuildLabel(name, s.pkg.Name)
		target = s.state.Graph.TargetOrDie(label)
	} else {
		target = getTargetPost(s, name)
	}

	outs := target.Outputs()
	ret := make(pyList, len(outs))
	for i, out := range outs {
		ret[i] = pyString(out)
	}
	return ret
}

// getNamedOuts gets the named outputs of a target
func getNamedOuts(s *scope, args []pyObject) pyObject {
	var target *core.BuildTarget
	if name := args[0].String(); core.LooksLikeABuildLabel(name) {
		label := core.ParseBuildLabel(name, s.pkg.Name)
		target = s.state.Graph.TargetOrDie(label)
	} else {
		target = getTargetPost(s, name)
	}

	var outs map[string][]string
	if target.IsFilegroup {
		outs = target.DeclaredNamedSources()
	} else {
		outs = target.DeclaredNamedOutputs()
	}

	ret := make(pyDict, len(outs))
	for k, v := range outs {
		list := make(pyList, len(v))
		for i, out := range v {
			list[i] = pyString(out)
		}
		ret[k] = list
	}
	return ret
}

// addLicence adds a licence to a target.
func addLicence(s *scope, args []pyObject) pyObject {
	target := getTargetPost(s, string(args[0].(pyString)))
	target.AddLicence(string(args[1].(pyString)))
	return None
}

// getLicences returns the licences for a single target.
func getLicences(s *scope, args []pyObject) pyObject {
	return fromStringList(getTargetPost(s, string(args[0].(pyString))).Licences)
}

// getCommand gets the command of a target, optionally for a configuration.
func getCommand(s *scope, args []pyObject) pyObject {
	target := getTargetPost(s, string(args[0].(pyString)))
	config := string(args[1].(pyString))
	if config != "" {
		return pyString(target.GetCommandConfig(config))
	}
	if len(target.Commands) > 0 {
		commands := pyDict{}
		for config, cmd := range target.Commands {
			commands[config] = pyString(cmd)
		}
		return commands
	}
	return pyString(target.Command)
}

// valueAsJSON returns a JSON-formatted string representation of a plz value.
func valueAsJSON(s *scope, args []pyObject) pyObject {
	js, err := json.Marshal(args[0])
	if err != nil {
		s.Error("Could not marshal object as JSON")
		return None
	}
	return pyString(js)
}

// setCommand sets the command of a target, optionally for a configuration.
func setCommand(s *scope, args []pyObject) pyObject {
	target := getTargetPost(s, string(args[0].(pyString)))
	config := string(args[1].(pyString))
	command := string(args[2].(pyString))
	if command == "" {
		target.Command = config
	} else {
		target.AddCommand(config, command)
	}
	return None
}

// selectFunc implements the select() builtin.
func selectFunc(s *scope, args []pyObject) pyObject {
	d, _ := asDict(args[0])
	var def pyObject

	// This is not really the same as Bazel's order-of-matching rules, but is at least deterministic.
	keys := d.Keys()
	for i := len(keys) - 1; i >= 0; i-- {
		k := keys[i]
		if k == "//conditions:default" || k == "default" {
			def = d[k]
		} else if selectTarget(s, s.parseLabelInContextPkg(k)).HasLabel("config:on") {
			return d[k]
		}
	}
	s.NAssert(def == nil, "None of the select() conditions matched")
	return def
}

// selectTarget returns the target to be used for a select() call.
// It panics appropriately if the target isn't built yet.
func selectTarget(s *scope, l core.BuildLabel) *core.BuildTarget {
	if s.pkg != nil && l.PackageName == s.pkg.Name {
		t := s.pkg.Target(l.Name)
		s.NAssert(t == nil, "Target %s in select() call has not been defined yet", l.Name)
		return t
	}
	return subincludeTarget(s, l)
}

// subrepo implements the subrepo() builtin that adds a new repository.
func subrepo(s *scope, args []pyObject) pyObject {
	const (
		NameArgIdx = iota
		DepArgIdx
		PathArgIdx
		ConfigArgIdx
		BazelCompatArgIdx
		ArchArgIdx
		PluginArgIdx
		PackageRootIdx
	)

	s.NAssert(s.pkg == nil, "Cannot create new subrepos in this scope")
	name := string(args[NameArgIdx].(pyString))
	dep := string(args[DepArgIdx].(pyString))

	// Root
	root := name
	var target *core.BuildTarget
	if dep != "" {
		// N.B. The target must be already registered on this package.
		target = s.pkg.TargetOrDie(s.parseLabelInPackage(dep, s.pkg).Name)
		if len(target.Outputs()) == 1 {
			root = filepath.Join(target.OutDir(), target.Outputs()[0])
		} else {
			// TODO(jpoole): perhaps this should be a fatal error?
			root = filepath.Join(target.OutDir(), name)
		}
	} else if args[PathArgIdx] != None {
		root = string(args[PathArgIdx].(pyString))
	}

	// Base name
	subrepoName := filepath.Join(s.pkg.Name, name)
	if args[PluginArgIdx].IsTruthy() {
		subrepoName = name
	}

	// State
	state := s.state.ForSubrepo(subrepoName, args[BazelCompatArgIdx].IsTruthy())

	// Arch
	isCrossCompile := s.pkg.Subrepo != nil && s.pkg.Subrepo.IsCrossCompile
	arch := cli.HostArch()
	if s.pkg.Subrepo != nil {
		arch = s.pkg.Subrepo.Arch
	}
	if args[ArchArgIdx] != None { // arg 5 is arch-string, for arch-subrepos.
		givenArch := string(args[ArchArgIdx].(pyString))
		if err := arch.UnmarshalFlag(givenArch); err != nil {
			log.Fatalf("Could not interpret architecture '%s' for subrepo '%s'", givenArch, name)
		}
		state = state.ForArch(arch)
		isCrossCompile = true
	} else if state.Arch != arch {
		state = state.ForArch(arch)
	}
	sr := core.NewSubrepo(state, s.pkg.SubrepoArchName(subrepoName), root, target, arch, isCrossCompile)
	if args[PackageRootIdx].IsTruthy() {
		sr.PackageRoot = args[PackageRootIdx].String()
	}

	// Typically this would be deferred until we have built the subrepo target and have its config available. As we
	// don't have a subrepo target, we can and should load it here.
	if target == nil {
		if err := sr.State.Initialise(sr); err != nil {
			log.Fatalf("Could not load subrepo config for %s: %v", sr.Name, err)
		}
	}

	if args[ConfigArgIdx].IsTruthy() {
		sr.AdditionalConfigFiles = append(sr.AdditionalConfigFiles, string(args[ConfigArgIdx].(pyString)))
	}

	if s.state.Config.Bazel.Compatibility && s.pkg.Name == "workspace" {
		sr.Name = s.pkg.SubrepoArchName(name)
	}

	log.Debug("Registering subrepo %s in package %s", sr.Name, s.pkg.Label())
	s.state.Graph.MaybeAddSubrepo(sr)
	return pyString("///" + sr.Name)
}

// breakpoint implements an interactive debugger for the breakpoint() builtin
func breakpoint(s *scope, args []pyObject) pyObject {
	if !s.state.EnableBreakpoints {
		log.Warningf("Skipping breakpoint. Use --debug to enable breakpoints.")
		return None
	}
	// Take this mutex to ensure only one debugger runs at a time
	s.interpreter.breakpointMutex.Lock()
	defer s.interpreter.breakpointMutex.Unlock()
	fmt.Printf("breakpoint() encountered in %s, entering interactive debugger...\n", s.pkg.Filename)
	// This is a small hack to get the return value back from an ident statement, which
	// is normally not available since we don't have implicit returns.
	interpretStatements := func(stmts []*Statement) (ret pyObject, err error) {
		if len(stmts) == 1 && stmts[0].Ident != nil {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("%s", r)
				}
			}()
			return s.interpretIdentStatement(stmts[0].Ident), nil
		}
		return s.interpreter.interpretStatements(s, stmts)
	}
	for {
		prompt := promptui.Prompt{
			Label: "plz",
			Validate: func(input string) error {
				_, err := s.interpreter.parser.ParseData([]byte(input), "<stdin>")
				return err
			},
		}
		if input, err := prompt.Run(); err != nil {
			if err == io.EOF || err.Error() == "^D" {
				break
			} else if err.Error() != "^C" {
				log.Error("%s", err)
			}
		} else if input == "exit" {
			break
		} else if stmts, err := s.interpreter.parser.ParseData([]byte(input), "<stdin>"); err != nil {
			log.Error("Syntax error: %s", err)
		} else if ret, err := interpretStatements(stmts); err != nil {
			log.Error("%s", err)
		} else if ret != nil && ret != None {
			fmt.Printf("%s\n", ret)
		} else {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("Debugger exited, continuing...\n")
	return None
}

func isSemver(s *scope, args []pyObject) pyObject {
	// semver.NewVersion is insufficiently strict for a validation function, since it coerces
	// semver-ish strings (e.g. "1.2") into semvers ("1.2.0"); semver.StrictNewVersion is slightly
	// too strict, since it doesn't allow the commonly-used leading "v". Stripping any leading "v"
	// and using semver.StrictNewVersion is a decent compromise
	_, err := semver.StrictNewVersion(strings.TrimPrefix(string(args[0].(pyString)), "v"))
	return newPyBool(err == nil)
}

func semverCheck(s *scope, args []pyObject) pyObject {
	v, err := semver.NewVersion(string(args[0].(pyString)))
	if err != nil {
		s.Error("failed to parse version: %v", err)

		return newPyBool(false)
	}

	c, err := semver.NewConstraint(string(args[1].(pyString)))
	if err != nil {
		s.Error("failed to parse constraint: %v", err)

		return newPyBool(false)
	}

	return newPyBool(c.Check(v))
}
