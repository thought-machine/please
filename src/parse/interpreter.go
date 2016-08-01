// Rule parser using PyPy. To build this you need PyPy installed, but the stock one
// that comes with Ubuntu will not work since it doesn't include shared libraries.
// For now we suggest fetching the upstream packages from pypy.org. Other distros
// might work fine though.
// On OSX installing through Homebrew should be fine.
//
// The interface to PyPy is done through cgo and cffi. This means that we need to write very little
// actual C code; nearly all of it is in interpreter.h and is just declarations. What remains in
// interpreter.c is essentially just glue to handle limitations of cgo and the way we're using
// callbacks etc.
// The setup isn't actually extremely complex but some care is needed; it's relatively rare to need
// to modify it (generally only when adding new properties to build targets) but when you do you
// must make sure this file, defs.h / interpreter.h and cffi/please_parser.py all agree about struct
// definitions etc. Bad Things will happen if you do not.

package parse

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/kardianos/osext"
	"gopkg.in/op/go-logging.v1"

	"core"
)

/*
#cgo CFLAGS: --std=c99 -Werror
#cgo !freebsd LDFLAGS: -ldl
#include "interpreter.h"
*/
import "C"

var log = logging.MustGetLogger("parse")

const subincludePackage = "_remote"

// Communicated back from PyPy to indicate that a parse has been deferred because
// we need to wait for another target to build.
const pyDeferParse = "_DEFER_"

// To ensure we only initialise once.
var initializeOnce sync.Once

// Code to initialise the Python interpreter.
func initializeInterpreter(config *core.Configuration) {
	log.Debug("Initialising interpreter...")

	// PyPy becomes very unhappy if Go schedules it to a different OS thread during
	// its initialisation. Force it to stay on this one thread for now.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Set the hash seed for Python dicts / sets; there isn't a DoS security concern in our context,
	// and it's much more useful to us that they are consistent between runs since it's not that hard
	// to accidentally write rules that are nondeterministic via {}.items() etc.
	os.Setenv("PYTHONHASHSEED", "42")

	// If an engine has been explicitly set, by flag or config, we honour it here.
	if config.Please.ParserEngine != "" {
		if !initialiseInterpreter(config.Please.ParserEngine) {
			log.Fatalf("Failed to initialise requested parser engine [%s]", config.Please.ParserEngine)
		}
	} else {
		// Okay, now try the standard fallbacks.
		// The python3 interpreter isn't ready yet, so don't try that.
		if !initialiseInterpreter("pypy") && !initialiseInterpreter("python2") {
			log.Fatalf("Can't initialise any Please parser engine. Please is putting itself out of its misery.")
		}
	}
	setConfigValue("PLZ_VERSION", config.Please.Version)
	setConfigValue("GO_VERSION", config.Go.GoVersion)
	setConfigValue("GO_TEST_TOOL", config.Go.TestTool)
	setConfigValue("GOPATH", config.Go.GoPath)
	setConfigValue("PIP_TOOL", config.Python.PipTool)
	setConfigValue("PEX_TOOL", config.Python.PexTool)
	setConfigValue("DEFAULT_PYTHON_INTERPRETER", config.Python.DefaultInterpreter)
	setConfigValue("PYTHON_MODULE_DIR", config.Python.ModuleDir)
	setConfigValue("PYTHON_DEFAULT_PIP_REPO", config.Python.DefaultPipRepo)
	setConfigValue("USE_PYPI", pythonBool(config.Python.UsePyPI))
	setConfigValue("JAVAC_TOOL", config.Java.JavacTool)
	setConfigValue("JARCAT_TOOL", config.Java.JarCatTool)
	setConfigValue("JUNIT_RUNNER", config.Java.JUnitRunner)
	setConfigValue("DEFAULT_TEST_PACKAGE", config.Java.DefaultTestPackage)
	setConfigValue("PLEASE_MAVEN_TOOL", config.Java.PleaseMavenTool)
	setConfigValue("JAVA_SOURCE_LEVEL", config.Java.SourceLevel)
	setConfigValue("JAVA_TARGET_LEVEL", config.Java.TargetLevel)
	setConfigValue("JAVAC_FLAGS", config.Java.JavacFlags)
	setConfigValue("JAVAC_TEST_FLAGS", config.Java.JavacTestFlags)
	setConfigValue("DEFAULT_MAVEN_REPO", config.Java.DefaultMavenRepo)
	setConfigValue("CC_TOOL", config.Cpp.CCTool)
	setConfigValue("LD_TOOL", config.Cpp.LdTool)
	setConfigValue("AR_TOOL", config.Cpp.ArTool)
	setConfigValue("DEFAULT_OPT_CFLAGS", config.Cpp.DefaultOptCflags)
	setConfigValue("DEFAULT_DBG_CFLAGS", config.Cpp.DefaultDbgCflags)
	setConfigValue("DEFAULT_LDFLAGS", config.Cpp.DefaultLdflags)
	setConfigValue("DEFAULT_NAMESPACE", config.Cpp.DefaultNamespace)
	setConfigValue("OS", runtime.GOOS)
	setConfigValue("ARCH", runtime.GOARCH)
	for _, language := range config.Proto.Language {
		setConfigValue("PROTO_LANGUAGES", language)
	}
	setConfigValue("PROTOC_TOOL", config.Proto.ProtocTool)
	setConfigValue("PROTOC_GO_PLUGIN", config.Proto.ProtocGoPlugin)
	setConfigValue("GRPC_PYTHON_PLUGIN", config.Proto.GrpcPythonPlugin)
	setConfigValue("GRPC_JAVA_PLUGIN", config.Proto.GrpcJavaPlugin)
	setConfigValue("GRPC_CC_PLUGIN", config.Proto.GrpcCCPlugin)
	setConfigValue("PROTOC_VERSION", config.Proto.ProtocVersion)
	setConfigValue("PROTO_PYTHON_DEP", config.Proto.PythonDep)
	setConfigValue("PROTO_JAVA_DEP", config.Proto.JavaDep)
	setConfigValue("PROTO_GO_DEP", config.Proto.GoDep)
	setConfigValue("PROTO_PYTHON_PACKAGE", config.Proto.PythonPackage)
	setConfigValue("GRPC_VERSION", config.Proto.GrpcVersion)
	setConfigValue("GRPC_PYTHON_DEP", config.Proto.PythonGrpcDep)
	setConfigValue("GRPC_JAVA_DEP", config.Proto.JavaGrpcDep)
	setConfigValue("GRPC_GO_DEP", config.Proto.GoGrpcDep)
	setConfigValue("BAZEL_COMPATIBILITY", pythonBool(config.Bazel.Compatibility))

	// Load all the builtin rules
	log.Debug("Loading builtin build rules...")
	dir, _ := AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		loadBuiltinRules(filename)
	}
	loadSubincludePackage()
	log.Debug("Interpreter ready")
}

// pythonBool returns the representation of a bool we're going to send to Python.
// We use strings to avoid having to do a different callback, but using the empty string for
// false means normal truth checks work fine :)
func pythonBool(b bool) string {
	if b {
		return "true"
	}
	return ""
}

func initialiseInterpreter(engine string) bool {
	if strings.HasPrefix(engine, "/") {
		return initialiseInterpreterFrom(engine)
	}
	executableDir, err := osext.ExecutableFolder()
	if err != nil {
		log.Error("Can't determine current executable: %s", err)
		return false
	}
	return initialiseInterpreterFrom(path.Join(executableDir, fmt.Sprintf("libplease_parser_%s.%s", engine, libExtension())))
}

func initialiseInterpreterFrom(enginePath string) bool {
	if !core.PathExists(enginePath) {
		return false
	}
	log.Debug("Attempting to load engine from %s", enginePath)
	cEnginePath := C.CString(enginePath)
	defer C.free(unsafe.Pointer(cEnginePath))
	result := C.InitialiseInterpreter(cEnginePath)
	if result != 0 {
		// Low level of logging because it's allowable to fail on libplease_parser_pypy, which we try first.
		log.Notice("Failed to initialise interpreter from %s: %s", enginePath, C.GoString(C.dlerror()))
		return false
	}
	log.Info("Using parser engine from %s", enginePath)
	return true
}

// libExtension returns the typical extension of shared objects on the current platform.
func libExtension() string {
	if runtime.GOOS == "darwin" {
		return "dylib"
	}
	return "so"
}

func setConfigValue(name string, value string) {
	cName := C.CString(name)
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cName))
	defer C.free(unsafe.Pointer(cValue))
	C.SetConfigValue(cName, cValue)
}

func loadBuiltinRules(path string) {
	data := loadAsset(path)
	defer C.free(unsafe.Pointer(data))
	cPackageName := C.CString(path)
	defer C.free(unsafe.Pointer(cPackageName))
	if result := C.GoString(C.ParseCode(data, cPackageName, 0)); result != "" {
		// This obviously shouldn't happen, because we control all the builtin rules.
		// It's here for developing rules in Please in case one makes a mistake :)
		log.Fatalf("Failed to interpret builtin build rules from %s: %s", path, result)
	}
}

func loadAsset(path string) *C.char {
	data := MustAsset(path)
	// well this is pretty inefficient... we end up with three copies of the data for no
	// really good reason.
	return C.CString(string(data))
}

func loadSubincludePackage() {
	pkg := core.NewPackage(subincludePackage)
	// Set up a builtin package for remote subincludes.
	cPackageName := C.CString(pkg.Name)
	C.ParseCode(nil, cPackageName, sizep(pkg))
	C.free(unsafe.Pointer(cPackageName))
	core.State.Graph.AddPackage(pkg)
}

// sizet converts a build target to a C.size_t.
func sizet(t *core.BuildTarget) C.size_t { return C.size_t(uintptr(unsafe.Pointer(t))) }

// sizep converts a package to a C.size_t
func sizep(p *core.Package) C.size_t { return C.size_t(uintptr(unsafe.Pointer(p))) }

// unsizet converts a C.size_t back to a *BuildTarget.
func unsizet(u uintptr) *core.BuildTarget { return (*core.BuildTarget)(unsafe.Pointer(u)) }

// unsizep converts a C.size_t back to a *Package
func unsizep(u uintptr) *core.Package { return (*core.Package)(unsafe.Pointer(u)) }

// parsePackageFile parses a single BUILD file.
// It returns true if parsing is deferred and waiting on other build actions, false otherwise on success
// and will panic on errors.
func parsePackageFile(state *core.BuildState, filename string, pkg *core.Package) bool {
	log.Debug("Parsing package file %s", filename)
	start := time.Now()
	initializeOnce.Do(func() { initializeInterpreter(state.Config) })
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// TODO(pebers): It seems like we should be calling C.pypy_attach_thread here once per OS thread.
	//               That only seems to introduce problems though and not solve them; not sure if that is
	//               because we are doing thread-unsafe things in our parser, more go/c/pypy interface
	//               issues or something more mysterious. Regardless, it would be nice to understand
	//               more what's going on there and see if we can solve - I'm not sure we really have
	//               multithreaded parsing without it.
	cFilename := C.CString(filename)
	cPackageName := C.CString(pkg.Name)
	defer C.free(unsafe.Pointer(cFilename))
	defer C.free(unsafe.Pointer(cPackageName))
	ret := C.GoString(C.ParseFile(cFilename, cPackageName, sizep(pkg)))
	if ret != "" && ret != pyDeferParse {
		panic(fmt.Sprintf("Failed to parse file %s: %s", filename, ret))
	}
	log.Debug("Parsed package file %s in %0.3f seconds", filename, time.Since(start).Seconds())
	return ret == pyDeferParse
}

// IsValidTargetName returns true if the given name is valid in a package.
// This is provided to help error handling on the Python side.
//export IsValidTargetName
func IsValidTargetName(name *C.char) bool {
	_, err := core.TryNewBuildLabel("test", C.GoString(name))
	return err == nil
}

//export AddTarget
func AddTarget(pkgPtr uintptr, cName, cCmd, cTestCmd *C.char, binary, test, needsTransitiveDeps,
	outputIsComplete, containerise, noTestOutput, testOnly, stamp bool,
	flakiness, buildTimeout, testTimeout int, cBuildingDescription *C.char) (ret C.size_t) {
	buildingDescription := ""
	if cBuildingDescription != nil {
		buildingDescription = C.GoString(cBuildingDescription)
	}
	return sizet(addTarget(pkgPtr, C.GoString(cName), C.GoString(cCmd), C.GoString(cTestCmd),
		binary, test, needsTransitiveDeps, outputIsComplete, containerise, noTestOutput,
		testOnly, stamp, flakiness, buildTimeout, testTimeout, buildingDescription))
}

// addTarget adds a new build target to the graph.
// Separated from AddTarget to make it possible to test (since you can't mix cgo and go test).
func addTarget(pkgPtr uintptr, name, cmd, testCmd string, binary, test, needsTransitiveDeps,
	outputIsComplete, containerise, noTestOutput, testOnly, stamp bool,
	flakiness, buildTimeout, testTimeout int, buildingDescription string) *core.BuildTarget {
	pkg := unsizep(pkgPtr)
	target := core.NewBuildTarget(core.NewBuildLabel(pkg.Name, name))
	target.IsBinary = binary
	target.IsTest = test
	target.NeedsTransitiveDependencies = needsTransitiveDeps
	target.OutputIsComplete = outputIsComplete
	target.Containerise = containerise
	target.NoTestOutput = noTestOutput
	target.TestOnly = testOnly
	target.Flakiness = flakiness
	target.BuildTimeout = buildTimeout
	target.TestTimeout = testTimeout
	target.Stamp = stamp
	// Automatically label containerised tests.
	if containerise {
		target.AddLabel("container")
	}
	// Automatically label flaky tests.
	if flakiness > 0 {
		target.AddLabel("flaky")
	}
	if binary {
		target.AddLabel("bin")
	}
	if buildingDescription != "" {
		target.BuildingDescription = buildingDescription
	}
	target.Command = cmd
	target.TestCommand = testCmd
	if _, present := pkg.Targets[name]; present {
		// NB. Not logged as an error because Python is now allowed to catch it.
		//     It will turn into an error later if the exception is not caught.
		log.Notice("Duplicate build target in %s: %s", pkg.Name, name)
		return nil
	}
	pkg.Targets[name] = target
	if core.State.Graph.Package(pkg.Name) != nil {
		// Package already added, so we're probably in a post-build function. Add target directly to graph now.
		log.Debug("Adding new target %s directly to graph", target.Label)
		core.State.Graph.AddTarget(target)
	}
	return target
}

//export SetPreBuildFunction
func SetPreBuildFunction(callback uintptr, cBytecode *C.char, cTarget uintptr) {
	target := unsizet(cTarget)
	target.PreBuildFunction = callback
	hash := sha1.Sum([]byte(C.GoString(cBytecode)))
	target.PreBuildHash = hash[:]
}

//export SetPostBuildFunction
func SetPostBuildFunction(callback uintptr, cBytecode *C.char, cTarget uintptr) {
	target := unsizet(cTarget)
	target.PostBuildFunction = callback
	hash := sha1.Sum([]byte(C.GoString(cBytecode)))
	target.PostBuildHash = hash[:]
}

//export AddDependency
func AddDependency(cPackage uintptr, cTarget *C.char, cDep *C.char, exported bool) *C.char {
	target, err := getTargetPost(cPackage, cTarget)
	if err != nil {
		return C.CString(err.Error())
	}
	dep, err := core.TryParseBuildLabel(C.GoString(cDep), target.Label.PackageName)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddMaybeExportedDependency(dep, exported)
	// Note that here we're in a post-build function so we must call this explicitly
	// (in other callbacks it's handled after the package parses all at once).
	core.State.Graph.AddDependency(target.Label, dep)
	return nil
}

//export AddOutputPost
func AddOutputPost(cPackage uintptr, cTarget *C.char, cOut *C.char) *C.char {
	target, err := getTargetPost(cPackage, cTarget)
	if err != nil {
		return C.CString(err.Error())
	}
	out := C.GoString(cOut)
	pkg := unsizep(cPackage)
	pkg.RegisterOutput(out, target)
	target.AddOutput(out)
	return nil
}

//export AddLicencePost
func AddLicencePost(cPackage uintptr, cTarget *C.char, cLicence *C.char) *C.char {
	target, err := getTargetPost(cPackage, cTarget)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddLicence(C.GoString(cLicence))
	return nil
}

//export SetCommand
func SetCommand(cPackage uintptr, cTarget *C.char, cConfigOrCommand *C.char, cCommand *C.char) *C.char {
	target, err := getTargetPost(cPackage, cTarget)
	if err != nil {
		return C.CString(err.Error())
	}
	command := C.GoString(cCommand)
	if command == "" {
		target.Command = C.GoString(cConfigOrCommand)
	} else {
		target.AddCommand(C.GoString(cConfigOrCommand), command)
	}
	// It'd be nice if we could ensure here that we're in the pre-build function
	// but not the post-build function which is too late to have any effect.
	// OTOH while it's ineffective it shouldn't cause any trouble trying it either...
	return nil
}

// Called by above to get a target from the current package.
// Returns an error if the target is not in the current package or has already been built.
func getTargetPost(cPackage uintptr, cTarget *C.char) (*core.BuildTarget, error) {
	pkg := unsizep(cPackage)
	name := C.GoString(cTarget)
	target, present := pkg.Targets[name]
	if !present {
		return nil, fmt.Errorf("Unknown build target %s in %s", name, pkg.Name)
	}
	// It'd be cheating to try to modify targets that're already built.
	// Prohibit this because it'd likely end up with nasty race conditions.
	if target.State() >= core.Built {
		return nil, fmt.Errorf("Attempted to modify target %s, but it's already built", target.Label)
	}
	return target, nil
}

//export AddSource
func AddSource(cTarget uintptr, cSource *C.char) *C.char {
	target := unsizet(cTarget)
	source, err := parseSource(C.GoString(cSource), target.Label.PackageName, true)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddSource(source)
	return nil
}

// Parses an incoming source label as either a file or a build label.
// Identifies if the file is owned by this package and returns an error if not.
func parseSource(src, packageName string, systemAllowed bool) (core.BuildInput, error) {
	if core.LooksLikeABuildLabel(src) {
		return core.TryParseBuildLabel(src, packageName)
	} else if src == "" {
		return nil, fmt.Errorf("Empty source path (in package %s)", packageName)
	} else if strings.Contains(src, "../") {
		return nil, fmt.Errorf("'%s' (in package %s) is an invalid path; build target paths can't contain ../", src, packageName)
	} else if src[0] == '/' {
		if !systemAllowed {
			return nil, fmt.Errorf("'%s' (in package %s) is an absolute path; that's not allowed.", src, packageName)
		}
		return core.SystemFileLabel{Path: src}, nil
	} else if strings.Contains(src, "/") {
		// Target is in a subdirectory, check nobody else owns that.
		for dir := path.Dir(path.Join(packageName, src)); dir != packageName && dir != "."; dir = path.Dir(dir) {
			if isPackage(dir) {
				return nil, fmt.Errorf("Package %s tries to use file %s, but that belongs to another package (%s).", packageName, src, dir)
			}
		}
	}
	return core.FileLabel{File: src, Package: packageName}, nil
}

//export AddNamedSource
func AddNamedSource(cTarget uintptr, cName *C.char, cSource *C.char) *C.char {
	target := unsizet(cTarget)
	source, err := parseSource(C.GoString(cSource), target.Label.PackageName, false)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddNamedSource(C.GoString(cName), source)
	return nil
}

//export AddCommand
func AddCommand(cTarget uintptr, cConfig *C.char, cCommand *C.char) *C.char {
	unsizet(cTarget).AddCommand(C.GoString(cConfig), C.GoString(cCommand))
	return nil
}

//export AddTestCommand
func AddTestCommand(cTarget uintptr, cConfig *C.char, cCommand *C.char) *C.char {
	unsizet(cTarget).AddTestCommand(C.GoString(cConfig), C.GoString(cCommand))
	return nil
}

//export AddData
func AddData(cTarget uintptr, cData *C.char) *C.char {
	target := unsizet(cTarget)
	data, err := parseSource(C.GoString(cData), target.Label.PackageName, false)
	if err != nil {
		return C.CString(err.Error())
	}
	target.Data = append(target.Data, data)
	if label := data.Label(); label != nil {
		target.AddDependency(*label)
	}
	return nil
}

//export AddOutput
func AddOutput(cTarget uintptr, cOutput *C.char) *C.char {
	target := unsizet(cTarget)
	target.AddOutput(C.GoString(cOutput))
	return nil
}

//export AddDep
func AddDep(cTarget uintptr, cDep *C.char) *C.char {
	target := unsizet(cTarget)
	dep, err := core.TryParseBuildLabel(C.GoString(cDep), target.Label.PackageName)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddDependency(dep)
	return nil
}

//export AddExportedDep
func AddExportedDep(cTarget uintptr, cDep *C.char) *C.char {
	target := unsizet(cTarget)
	dep, err := core.TryParseBuildLabel(C.GoString(cDep), target.Label.PackageName)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddMaybeExportedDependency(dep, true)
	return nil
}

//export AddTool
func AddTool(cTarget uintptr, cTool *C.char) *C.char {
	target := unsizet(cTarget)
	tool, err := core.TryParseBuildLabel(C.GoString(cTool), target.Label.PackageName)
	if err != nil {
		return C.CString(err.Error())
	}
	target.Tools = append(target.Tools, tool)
	target.AddDependency(tool)
	return nil
}

//export AddVis
func AddVis(cTarget uintptr, cVis *C.char) *C.char {
	target := unsizet(cTarget)
	vis := C.GoString(cVis)
	if vis == "PUBLIC" || (core.State.Config.Bazel.Compatibility && vis == "//visibility:public") {
		target.Visibility = append(target.Visibility, core.WholeGraph[0])
	} else {
		label, err := core.TryParseBuildLabel(vis, target.Label.PackageName)
		if err != nil {
			return C.CString(err.Error())
		}
		target.Visibility = append(target.Visibility, label)
	}
	return nil
}

//export AddLabel
func AddLabel(cTarget uintptr, cLabel *C.char) *C.char {
	target := unsizet(cTarget)
	target.AddLabel(C.GoString(cLabel))
	return nil
}

//export AddHash
func AddHash(cTarget uintptr, cHash *C.char) *C.char {
	target := unsizet(cTarget)
	target.Hashes = append(target.Hashes, C.GoString(cHash))
	return nil
}

//export AddLicence
func AddLicence(cTarget uintptr, cLicence *C.char) *C.char {
	target := unsizet(cTarget)
	target.AddLicence(C.GoString(cLicence))
	return nil
}

//export AddTestOutput
func AddTestOutput(cTarget uintptr, cTestOutput *C.char) *C.char {
	target := unsizet(cTarget)
	target.TestOutputs = append(target.TestOutputs, C.GoString(cTestOutput))
	return nil
}

//export AddRequire
func AddRequire(cTarget uintptr, cRequire *C.char) *C.char {
	target := unsizet(cTarget)
	target.Requires = append(target.Requires, C.GoString(cRequire))
	// Requirements are also implicit labels
	target.AddLabel(C.GoString(cRequire))
	return nil
}

//export AddProvide
func AddProvide(cTarget uintptr, cLanguage *C.char, cDep *C.char) *C.char {
	target := unsizet(cTarget)
	label, err := core.TryParseBuildLabel(C.GoString(cDep), target.Label.PackageName)
	if err != nil {
		return C.CString(err.Error())
	}
	target.AddProvide(C.GoString(cLanguage), label)
	return nil
}

//export SetContainerSetting
func SetContainerSetting(cTarget uintptr, cName, cValue *C.char) *C.char {
	target := unsizet(cTarget)
	if err := target.SetContainerSetting(strings.Replace(C.GoString(cName), "_", "", -1), C.GoString(cValue)); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

// GetIncludeFile is a callback to the interpreter that returns the path it
// should be opening in order to include_defs() a file.
// We use in-band signalling for some errors since C can't handle multiple return values :)
//export GetIncludeFile
func GetIncludeFile(cPackage uintptr, cLabel *C.char) *C.char {
	label := C.GoString(cLabel)
	if !strings.HasPrefix(label, "//") {
		return C.CString("__include_defs argument must be an absolute path (ie. start with //)")
	}
	relPath := strings.TrimLeft(label, "/")
	return C.CString(path.Join(core.RepoRoot, relPath))
}

// GetSubincludeFile is a callback to the interpreter that returns the path it
// should be opening in order to subinclude() a build target.
// We use in-band signalling for some errors since C can't handle multiple return values :)
//export GetSubincludeFile
func GetSubincludeFile(cPackage uintptr, cLabel *C.char) *C.char {
	return C.CString(getSubincludeFile(unsizep(cPackage), C.GoString(cLabel)))
}

func getSubincludeFile(pkg *core.Package, labelStr string) string {
	label := core.ParseBuildLabel(labelStr, pkg.Name)
	if label.PackageName == pkg.Name {
		return fmt.Sprintf("__Can't subinclude :%s in %s; can't subinclude local targets.", label.Name, pkg.Name)
	}
	pkgLabel := core.BuildLabel{PackageName: pkg.Name, Name: "all"}
	target := core.State.Graph.Target(label)
	if target == nil {
		// Might not have been parsed yet. Check for that first.
		if subincludePackage := core.State.Graph.Package(label.PackageName); subincludePackage == nil {
			if deferParse(label, pkg) {
				return pyDeferParse // Not an error, they'll just have to wait.
			}
			target = core.State.Graph.TargetOrDie(label) // Should be there now.
		} else {
			return fmt.Sprintf("__Failed to subinclude %s; package %s has no target by that name", label, label.PackageName)
		}
	} else if tmp := core.NewBuildTarget(pkgLabel); !tmp.CanSee(target) {
		return fmt.Sprintf("__Can't subinclude %s from %s due to visibility constraints", label, pkg.Name)
	} else if len(target.Outputs()) != 1 {
		return fmt.Sprintf("__Can't subinclude %s, subinclude targets must have exactly one output", label)
	} else if target.State() < core.Built {
		if deferParse(label, pkg) {
			return pyDeferParse // Again, they'll have to wait for this guy to build.
		}
	}
	pkg.RegisterSubinclude(target.Label)
	// Well if we made it to here it's actually ready to go, so tell them where to get it.
	return path.Join(target.OutDir(), target.Outputs()[0])
}

// runPreBuildFunction runs the pre-build function for a single target.
func runPreBuildFunction(pkg *core.Package, target *core.BuildTarget) error {
	cName := C.CString(target.Label.Name)
	defer C.free(unsafe.Pointer(cName))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f := C.size_t(uintptr(unsafe.Pointer(target.PreBuildFunction)))
	if result := C.GoString(C.RunPreBuildFunction(f, sizep(pkg), cName)); result != "" {
		return fmt.Errorf("Failed to run pre-build function for target %s: %s", target.Label.String(), result)
	}
	return nil
}

// runPostBuildFunction runs the post-build function for a single target.
func runPostBuildFunction(pkg *core.Package, target *core.BuildTarget, out string) error {
	cName := C.CString(target.Label.Name)
	cOutput := C.CString(out)
	defer C.free(unsafe.Pointer(cName))
	defer C.free(unsafe.Pointer(cOutput))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f := C.size_t(uintptr(unsafe.Pointer(target.PostBuildFunction)))
	if result := C.GoString(C.RunPostBuildFunction(f, sizep(pkg), cName, cOutput)); result != "" {
		return fmt.Errorf("Failed to run post-build function for target %s: %s", target.Label.String(), result)
	}
	return nil
}

// Unfortunately there doesn't seem to be any API to do this dynamically :(
var logLevelFuncs = map[logging.Level]func(format string, args ...interface{}){
	logging.CRITICAL: log.Fatalf,
	logging.ERROR:    log.Errorf,
	logging.WARNING:  log.Warning,
	logging.NOTICE:   log.Notice,
	logging.INFO:     log.Info,
	logging.DEBUG:    log.Debug,
}

//export Log
func Log(level int, cPackage uintptr, cMessage *C.char) {
	pkg := unsizep(cPackage)
	f, present := logLevelFuncs[logging.Level(level)]
	if !present {
		f = log.Errorf
	}
	f("//%s/BUILD: %s", pkg.Name, C.GoString(cMessage))
}

//export Glob
func Glob(cPackage *C.char, cIncludes **C.char, numIncludes int, cExcludes **C.char, numExcludes int, includeHidden bool) **C.char {
	packageName := C.GoString(cPackage)
	includes := cStringArrayToStringSlice(cIncludes, numIncludes, "")
	prefixedExcludes := cStringArrayToStringSlice(cExcludes, numExcludes, packageName)
	excludes := cStringArrayToStringSlice(cExcludes, numExcludes, "")
	filenames := globall(packageName, includes, prefixedExcludes, excludes, includeHidden)
	return stringSliceToCStringArray(filenames)
}

// stringSliceToCStringArray converts a Go slice of strings to a C array of char*'s.
// The returned array is terminated by a null pointer - the Python interpreter code will
// understand how to turn this back into Python strings.
func stringSliceToCStringArray(s []string) **C.char {
	// This is slightly hacky; we assume that sizeof(char*) == size of a uintptr in Go.
	// Presumably that should hold in most cases and is more portable than just hardcoding 8...
	const sz = int(unsafe.Sizeof(uintptr(0)))
	n := len(s) + 1
	ret := (**C.char)(C.malloc(C.size_t(sz * n)))
	sl := (*[1 << 30]*C.char)(unsafe.Pointer(ret))[:n:n]
	for i, x := range s {
		sl[i] = C.CString(x)
	}
	sl[n-1] = nil
	return ret
}

// cStringArrayToStringSlice converts a C array of char*'s to a Go slice of strings.
func cStringArrayToStringSlice(a **C.char, n int, prefix string) []string {
	ret := make([]string, n)
	// slightly scary incantation found on an internet
	sl := (*[1 << 30]*C.char)(unsafe.Pointer(a))[:n:n]
	for i, s := range sl {
		ret[i] = path.Join(prefix, C.GoString(s))
	}
	return ret
}

//export GetLabels
func GetLabels(cPackage uintptr, cTarget *C.char, cPrefix *C.char) **C.char {
	target, err := getTargetPost(cPackage, cTarget)
	if err != nil {
		log.Fatalf("%s", err) // TODO(pebers): report proper errors here and below
	}
	prefix := C.GoString(cPrefix)
	if target.State() != core.Building {
		log.Fatalf("get_labels called for %s incorrectly; the only time this is safe to call is from its own pre-build function.", target.Label)
	}
	labels := map[string]bool{}
	var getLabels func(*core.BuildTarget)
	getLabels = func(target *core.BuildTarget) {
		for _, label := range target.Labels {
			if strings.HasPrefix(label, prefix) {
				labels[strings.TrimSpace(strings.TrimPrefix(label, prefix))] = true
			}
		}
		for _, dep := range target.Dependencies() {
			getLabels(dep)
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
	return stringSliceToCStringArray(ret)
}
