// Utilities for reading the Please config files.

package core

import (
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"gopkg.in/gcfg.v1"

	"cli"
)

// File name for the typical repo config - this is normally checked in
const ConfigFileName string = ".plzconfig"

// Architecture-specific config file which overrides the repo one. Also normally checked in if needed.
var ArchConfigFileName string = fmt.Sprintf(".plzconfig_%s_%s", runtime.GOOS, runtime.GOARCH)

// File name for the local repo config - this is not normally checked in and used to
// override settings on the local machine.
const LocalConfigFileName string = ".plzconfig.local"

// File name for the machine-level config - can use this to override things
// for a particular machine (eg. build machine with different caching behaviour).
const MachineConfigFileName = "/etc/plzconfig"

const TestContainerDocker = "docker"
const TestContainerNone = "none"

func readConfigFile(config *Configuration, filename string) error {
	log.Debug("Reading config from %s...", filename)
	if err := gcfg.ReadFileInto(config, filename); err != nil && os.IsNotExist(err) {
		return nil // It's not an error to not have the file at all.
	} else if gcfg.FatalOnly(err) != nil {
		return err
	} else if err != nil {
		log.Warning("Error in config file: %s", err)
	}
	return nil
}

// Reads a config file from the given locations, in order.
// Values are filled in by defaults initially and then overridden by each file in turn.
func ReadConfigFiles(filenames []string) (*Configuration, error) {
	config := DefaultConfiguration()
	for _, filename := range filenames {
		if err := readConfigFile(config, filename); err != nil {
			return config, err
		}
	}
	// Set default values for slices. These add rather than overwriting so we can't set
	// them upfront as we would with other config values.
	setDefault(&config.Please.BuildFileName, []string{"BUILD"})
	setDefault(&config.Build.Path, []string{"/usr/local/bin", "/usr/bin", "/bin"})
	setDefault(&config.Cover.FileExtension, []string{".go", ".py", ".java", ".js", ".cc", ".h", ".c"})
	setDefault(&config.Cover.ExcludeExtension, []string{".pb.go", "_pb2.py", ".pb.cc", ".pb.h", "_test.py", "_test.go", "_pb.go", "_bindata.go", "_test_main.cc"})
	setDefault(&config.Proto.Language, []string{"cc", "py", "java", "go"})

	// Default values for these guys depend on config.Please.Location.
	defaultPath(&config.Cache.DirCacheCleaner, config.Please.Location, "cache_cleaner")
	defaultPath(&config.Go.TestTool, config.Please.Location, "please_go_test")
	defaultPath(&config.Python.PexTool, config.Please.Location, "please_pex")
	defaultPath(&config.Java.JavacWorker, config.Please.Location, "javac_worker")
	defaultPath(&config.Java.JarCatTool, config.Please.Location, "jarcat")
	defaultPath(&config.Java.PleaseMavenTool, config.Please.Location, "please_maven")
	defaultPath(&config.Java.JUnitRunner, config.Please.Location, "junit_runner.jar")

	if (config.Cache.RpcPrivateKey == "") != (config.Cache.RpcPublicKey == "") {
		return config, fmt.Errorf("Must pass both rpcprivatekey and rpcpublickey properties for cache")
	}
	return config, nil
}

// setDefault sets a slice of strings in the config if the set one is empty.
func setDefault(conf *[]string, def []string) {
	if len(*conf) == 0 {
		*conf = def
	}
}

// defaultPath sets a variable to a location in a directory if it's not already set.
func defaultPath(conf *string, dir, file string) {
	if *conf == "" {
		*conf = path.Join(dir, file)
	}
}

func DefaultConfiguration() *Configuration {
	config := Configuration{}
	config.Please.Location = "~/.please"
	config.Please.SelfUpdate = true
	config.Please.DownloadLocation = "https://get.please.build"
	config.Please.Lang = "en_GB.UTF-8" // Not the language of the UI, the language passed to rules.
	config.Please.Nonce = "1402"       // Arbitrary nonce to invalidate config when needed.
	config.Build.Timeout = cli.Duration(10 * time.Minute)
	config.Build.Config = "opt"         // Optimised builds by default
	config.Build.FallbackConfig = "opt" // Optimised builds as a fallback on any target that doesn't have a matching one set
	config.Cache.HttpTimeout = cli.Duration(5 * time.Second)
	config.Cache.RpcTimeout = cli.Duration(5 * time.Second)
	config.Cache.Dir = ".plz-cache"
	config.Cache.DirCacheHighWaterMark = "10G"
	config.Cache.DirCacheLowWaterMark = "8G"
	config.Cache.Workers = runtime.NumCPU() + 2 // Mirrors the number of workers in please.go.
	config.Cache.RpcMaxMsgSize.UnmarshalFlag("200MiB")
	config.Metrics.PushFrequency = cli.Duration(400 * time.Millisecond)
	config.Metrics.PushTimeout = cli.Duration(500 * time.Millisecond)
	config.Test.Timeout = cli.Duration(10 * time.Minute)
	config.Test.DefaultContainer = TestContainerDocker
	config.Docker.DefaultImage = "ubuntu:trusty"
	config.Docker.AllowLocalFallback = false
	config.Docker.Timeout = cli.Duration(20 * time.Minute)
	config.Docker.ResultsTimeout = cli.Duration(20 * time.Second)
	config.Docker.RemoveTimeout = cli.Duration(20 * time.Second)
	config.Go.CgoCCTool = "gcc"
	config.Go.GoVersion = "1.6"
	config.Go.GoPath = "$TMP_DIR:$TMP_DIR/src:$TMP_DIR/third_party/go"
	config.Python.PipTool = "pip"
	config.Python.DefaultInterpreter = "python"
	config.Python.UsePyPI = true
	// Annoyingly pip on OSX doesn't seem to work with this flag (you get the dreaded
	// "must supply either home or prefix/exec-prefix" error). Goodness knows why *adding* this
	// flag - which otherwise seems exactly what we want - provokes that error, but the logic
	// of pip is rather a mystery to me.
	if runtime.GOOS != "darwin" {
		config.Python.PipFlags = "--isolated"
	}
	config.Java.DefaultTestPackage = ""
	config.Java.SourceLevel = "8"
	config.Java.TargetLevel = "8"
	config.Java.DefaultMavenRepo = "https://repo1.maven.org/maven2"
	config.Java.JavacFlags = "-Werror -Xlint:-options" // bootstrap class path warnings are pervasive without this.
	config.Cpp.CCTool = "gcc"
	config.Cpp.CppTool = "g++"
	config.Cpp.LdTool = "ld"
	config.Cpp.ArTool = "ar"
	config.Cpp.AsmTool = "nasm"
	config.Cpp.DefaultOptCflags = "--std=c99 -O3 -pipe -DNDEBUG -Wall -Wextra -Werror"
	config.Cpp.DefaultDbgCflags = "--std=c99 -g3 -pipe -DDEBUG -Wall -Wextra -Werror"
	config.Cpp.DefaultOptCppflags = "--std=c++11 -O3 -pipe -DNDEBUG -Wall -Wextra -Werror"
	config.Cpp.DefaultDbgCppflags = "--std=c++11 -g3 -pipe -DDEBUG -Wall -Wextra -Werror"
	config.Cpp.Coverage = true
	config.Proto.ProtocTool = "protoc"
	// We're using the most common names for these; typically gRPC installs the builtin plugins
	// as grpc_python_plugin etc.
	config.Proto.ProtocGoPlugin = "protoc-gen-go"
	config.Proto.GrpcPythonPlugin = "grpc_python_plugin"
	config.Proto.GrpcJavaPlugin = "protoc-gen-grpc-java"
	config.Proto.GrpcCCPlugin = "grpc_cpp_plugin"
	config.Proto.PythonDep = "//third_party/python:protobuf"
	config.Proto.JavaDep = "//third_party/java:protobuf"
	config.Proto.GoDep = "//third_party/go:protobuf"
	config.Proto.PythonGrpcDep = "//third_party/python:grpc"
	config.Proto.JavaGrpcDep = "//third_party/java:grpc-all"
	config.Proto.GoGrpcDep = "//third_party/go:grpc"
	return &config
}

type Configuration struct {
	Please struct {
		Version          semver.Version
		Location         string
		SelfUpdate       bool
		DownloadLocation string
		BuildFileName    []string
		BlacklistDirs    []string
		Lang             string
		PyPyLocation     []string
		ParserEngine     string
		Nonce            string
		NumThreads       int
		ExperimentalDir  string
	}
	Build struct {
		Timeout        cli.Duration
		Path           []string
		Config         string
		FallbackConfig string
	}
	BuildConfig map[string]string
	Cache       struct {
		Workers               int
		Dir                   string
		DirCacheCleaner       string
		DirCacheHighWaterMark string
		DirCacheLowWaterMark  string
		HttpUrl               string
		HttpWriteable         bool
		HttpTimeout           cli.Duration
		RpcUrl                string
		RpcWriteable          bool
		RpcTimeout            cli.Duration
		RpcPublicKey          string
		RpcPrivateKey         string
		RpcCACert             string
		RpcSecure             bool
		RpcMaxMsgSize         cli.ByteSize
	}
	Metrics struct {
		PushGatewayURL string
		PushFrequency  cli.Duration
		PushTimeout    cli.Duration
	}
	CustomMetricLabels map[string]string
	Test               struct {
		Timeout          cli.Duration
		DefaultContainer ContainerImplementation
	}
	Cover struct {
		FileExtension    []string
		ExcludeExtension []string
	}
	Docker struct {
		DefaultImage       string
		AllowLocalFallback bool
		Timeout            cli.Duration
		ResultsTimeout     cli.Duration
		RemoveTimeout      cli.Duration
		RunArgs            []string
	}
	Gc struct {
		Keep      []BuildLabel
		KeepLabel []string
	}
	Go struct {
		GoVersion string
		GoRoot    string
		TestTool  string
		GoPath    string
		CgoCCTool string
	}
	Python struct {
		PipTool            string
		PipFlags           string
		PexTool            string
		DefaultInterpreter string
		ModuleDir          string
		DefaultPipRepo     string
		UsePyPI            bool
	}
	Java struct {
		JavacTool          string
		JavacWorker        string
		JarTool            string
		JarCatTool         string
		PleaseMavenTool    string
		JUnitRunner        string
		DefaultTestPackage string
		SourceLevel        string
		TargetLevel        string
		JavacFlags         string
		JavacTestFlags     string
		DefaultMavenRepo   string
	}
	Cpp struct {
		CCTool             string
		CppTool            string
		LdTool             string
		ArTool             string
		AsmTool            string
		LinkWithLdTool     bool
		DefaultOptCflags   string
		DefaultDbgCflags   string
		DefaultOptCppflags string
		DefaultDbgCppflags string
		DefaultLdflags     string
		DefaultNamespace   string
		Coverage           bool
	}
	Proto struct {
		ProtocTool       string
		ProtocGoPlugin   string
		GrpcPythonPlugin string
		GrpcJavaPlugin   string
		GrpcCCPlugin     string
		Language         []string
		PythonDep        string
		JavaDep          string
		GoDep            string
		PythonGrpcDep    string
		JavaGrpcDep      string
		GoGrpcDep        string
		PythonPackage    string
	}
	Licences struct {
		Accept []string
		Reject []string
	}
	Aliases map[string]string
	Bazel   struct {
		Compatibility bool
	}
}

func (config *Configuration) Hash() []byte {
	h := sha1.New()
	// These fields are the ones that need to be in the general hash; other things will be
	// picked up by relevant rules (particularly tool paths etc).
	// Note that container settings are handled separately.
	for _, f := range config.Please.BuildFileName {
		h.Write([]byte(f))
	}
	h.Write([]byte(config.Please.Lang))
	h.Write([]byte(config.Please.Nonce))
	for _, p := range config.Build.Path {
		h.Write([]byte(p))
	}
	for _, l := range config.Licences.Reject {
		h.Write([]byte(l))
	}
	return h.Sum(nil)
}

// ContainerisationHash returns the hash of the containerisation part of the config.
func (config *Configuration) ContainerisationHash() []byte {
	h := sha1.New()
	encoder := gob.NewEncoder(h)
	if err := encoder.Encode(config.Docker); err != nil {
		panic(err)
	}
	return h.Sum(nil)
}

// ApplyOverrides applies a set of overrides to the config.
// The keys of the given map are dot notation for the config setting.
func (config *Configuration) ApplyOverrides(overrides map[string]string) error {
	match := func(s1 string) func(string) bool {
		return func(s2 string) bool {
			return strings.ToLower(s2) == s1
		}
	}
	elem := reflect.ValueOf(config).Elem()
	for k, v := range overrides {
		split := strings.Split(strings.ToLower(k), ".")
		if len(split) != 2 {
			return fmt.Errorf("Bad option format: %s", k)
		}
		field := elem.FieldByNameFunc(match(split[0]))
		if !field.IsValid() {
			return fmt.Errorf("Unknown config field: %s", split[0])
		} else if field.Kind() != reflect.Struct {
			return fmt.Errorf("Unsettable config field: %s", split[0])
		}
		field = field.FieldByNameFunc(match(split[1]))
		if !field.IsValid() {
			return fmt.Errorf("Unknown config field: %s", split[1])
		}
		switch field.Kind() {
		case reflect.String:
			field.Set(reflect.ValueOf(v))
		case reflect.Bool:
			v = strings.ToLower(v)
			// Mimics the set of truthy things gcfg accepts in our config file.
			field.SetBool(v == "true" || v == "yes" || v == "on" || v == "1")
		case reflect.Int:
			i, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("Invalid value for an integer field: %s", v)
			}
			field.Set(reflect.ValueOf(i))
		case reflect.Int64:
			var d cli.Duration
			if err := d.UnmarshalText([]byte(v)); err != nil {
				return fmt.Errorf("Invalid value for a duration field: %s", v)
			}
			field.Set(reflect.ValueOf(d))
		case reflect.Slice:
			// We only have to worry about slices of strings. Comma-separated values are accepted.
			field.Set(reflect.ValueOf(strings.Split(v, ",")))
		default:
			return fmt.Errorf("Can't override config field %s (is %s)", k, field.Kind())
		}
	}
	return nil
}

// ContainerImplementation is an enumerated type for the container engine we'd use.
type ContainerImplementation string

func (ci *ContainerImplementation) UnmarshalText(text []byte) error {
	if ContainerImplementation(text) == ContainerImplementationNone || ContainerImplementation(text) == ContainerImplementationDocker {
		*ci = ContainerImplementation(text)
		return nil
	}
	return fmt.Errorf("Unknown container implementation: %s", string(text))
}

const (
	ContainerImplementationNone   ContainerImplementation = "none"
	ContainerImplementationDocker ContainerImplementation = "docker"
)
