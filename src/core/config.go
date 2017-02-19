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
	setDefault(&config.Proto.Language, []string{"cc", "py", "java", "go", "js"})

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
	config.Cpp.DefaultOptCflags = "--std=c99 -O3 -pipe -DNDEBUG -Wall -Werror"
	config.Cpp.DefaultDbgCflags = "--std=c99 -g3 -pipe -DDEBUG -Wall -Werror"
	config.Cpp.DefaultOptCppflags = "--std=c++11 -O3 -pipe -DNDEBUG -Wall -Werror"
	config.Cpp.DefaultDbgCppflags = "--std=c++11 -g3 -pipe -DDEBUG -Wall -Werror"
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
	config.Proto.JsDep = ""
	config.Proto.PythonGrpcDep = "//third_party/python:grpc"
	config.Proto.JavaGrpcDep = "//third_party/java:grpc-all"
	config.Proto.GoGrpcDep = "//third_party/go:grpc"
	config.Bazel.Compatibility = usingBazelWorkspace
	return &config
}

type Configuration struct {
	Please struct {
		Version          semver.Version `help:"Defines the version of plz that this repo is supposed to use currently. If it's not present or the version matches the currently running version no special action is taken; otherwise if SelfUpdate is set Please will attempt to download an appropriate version, otherwise it will issue a warning and continue.\n\nNote that if this is not set, you can run plz update to update to the latest version available on the server."`
		Location         string         `help:"Defines the directory Please is installed into.\nDefaults to ~/.please but you might want it to be somewhere else if you're installing via another method (e.g. the debs and install script still use /opt/please)."`
		SelfUpdate       bool           `help:"Sets whether plz will attempt to update itself when the version set in the config file is different."`
		DownloadLocation cli.URL        `help:"Defines the location to download Please from when self-updating. Defaults to the Please web server, but you can point it to some location of your own if you prefer to keep traffic within your network or use home-grown versions."`
		BuildFileName    []string       `help:"Sets the names that Please uses instead of BUILD for its build files.\nFor clarity the documentation refers to them simply as BUILD files but you could reconfigure them here to be something else.\nOne case this can be particularly useful is in cases where you have a subdirectory named build on a case-insensitive file system like HFS+."`
		BlacklistDirs    []string       `help:"Directories to blacklist when recursively searching for BUILD files (e.g. when using plz build ... or similar).\nThis is generally useful when you have large directories within your repo that don't need to be searched, especially things like node_modules that have come from external package managers."`
		Lang             string         `help:"Sets the language passed to build rules when building. This can be important for some tools (although hopefully not many) - we've mostly observed it with Sass."`
		ParserEngine     string
		Nonce            string `help:"This is an arbitrary string that is added to the hash of every build target. It provides a way to force a rebuild of everything when it's changed.\nWe will bump the default of this whenever we think it's required - although it's been a pretty long time now and we hope that'll continue."`
		NumThreads       int    `help:"Number of parallel build operations to run.\nIs overridden by the equivalent command-line flag, if that's passed."`
		ExperimentalDir  string
	}
	Display struct {
		UpdateTitle bool
	}
	Build struct {
		Timeout        cli.Duration `help:"Default timeout for Dockerised tests, in seconds. Default is 1200 (twenty minutes)."`
		Path           []string     `help:"The PATH variable that will be passed to the build processes.\nDefaults to /usr/local/bin:/usr/bin:/bin but of course can be modified if you need to get binaries from other locations."`
		Config         string       `help:"The build config to use when one is not chosen on the command line. Defaults to opt."`
		FallbackConfig string       `help:"The build config to use when one is chosen and a required target does not have one by the same name. Also defaults to opt."`
	}
	BuildConfig map[string]string
	Cache       struct {
		Workers               int
		Dir                   string       `help:"Sets the directory to use for the dir cache.\nThe default is .plz-cache, if set to the empty string the dir cache will be disabled."`
		DirCacheCleaner       string       `help:"The binary to use for cleaning the directory cache.\nDefaults to cache_cleaner in the plz install directory.\nCan also be set to the empty string to disable attempting to run it - note that this will of course lead to the dir cache growing without limit which may ruin your day if it fills your disk :)"`
		DirCacheHighWaterMark string       `help:"Starts cleaning the directory cache when it is over this number of bytes.\nCan also be given with human-readable suffixes like 10G, 200MB etc."`
		DirCacheLowWaterMark  string       `help:"When cleaning the directory cache, it's reduced to at most this size."`
		HttpUrl               cli.URL      `help:"Base URL of the HTTP cache.\nNot set to anything by default which means the cache will be disabled."`
		HttpWriteable         bool         `help:"If True this plz instance will write content back to the HTTP cache.\nBy default it runs in read-only mode."`
		HttpTimeout           cli.Duration `help:"Timeout for operations contacting the HTTP cache, in seconds."`
		RpcUrl                cli.URL      `help:"Base URL of the RPC cache.\nNot set to anything by default which means the cache will be disabled."`
		RpcWriteable          bool         `help:"If True this plz instance will write content back to the RPC cache.\nBy default it runs in read-only mode."`
		RpcTimeout            cli.Duration `help:"Timeout for operations contacting the RPC cache, in seconds."`
		RpcPublicKey          string
		RpcPrivateKey         string
		RpcCACert             string
		RpcSecure             bool
		RpcMaxMsgSize         cli.ByteSize `help:"Maximum size of a single message that we'll send to the RPC server.\nThis should agree with the server's limit, if it's higher the artifacts will be rejected.\nThe value is given as a byte size so can be suffixed with M, GB, KiB, etc."`
	}
	Metrics struct {
		PushGatewayURL cli.URL      `help:"The URL of the pushgateway to send metrics to."`
		PushFrequency  cli.Duration `help:"The frequency, in milliseconds, to push statistics at. Defaults to 100."`
		PushTimeout    cli.Duration
	}
	CustomMetricLabels map[string]string
	Test               struct {
		Timeout          cli.Duration
		DefaultContainer ContainerImplementation `help:"Sets the default type of containerisation to use for tests that are given container = True.\nCurrently the only option is 'docker' but we intend to add rkt support at some point."`
	}
	Cover struct {
		FileExtension    []string `help:"Extensions of files to consider for coverage.\nDefaults to a reasonably obvious set for the builtin rules including .go, .py, .java, etc."`
		ExcludeExtension []string `help:"Extensions of files to exclude from coverage.\nTypically this is for generated code; the default is to exclude protobuf extensions like .pb.go, _pb2.py, etc."`
	}
	Docker struct {
		DefaultImage       string `help:"The default image used for any test that doesn't specify another."`
		AllowLocalFallback bool   `help:"If True, will attempt to run the test locally if containerised running fails."`
		Timeout            cli.Duration
		ResultsTimeout     cli.Duration `help:"Timeout to wait when trying to retrieve results from inside the container. Default is 20 seconds."`
		RemoveTimeout      cli.Duration `help:"Timeout to wait when trying to remove a container after running a test. Defaults to 20 seconds."`
		RunArgs            []string     `help:"Arguments passed to docker run when running a test."`
	}
	Gc struct {
		Keep      []BuildLabel `help:"Marks targets that gc should always keep. Can include meta-targets such as //test/... and //docs:all."`
		KeepLabel []string     `help:"Defines a target label to be kept; for example, if you set this to go, no Go targets would ever be considered for deletion."`
	}
	Go struct {
		GoVersion string `help:"String identifying the version of the Go compiler.\nThis is only now really important for anyone targeting versions of Go earlier than 1.5 since some of the tool names have changed (6g and 6l became compile and link in Go 1.5).\nWe're pretty sure that targeting Go 1.4 works; we're not sure about 1.3 (never tried) but 1.2 certainly doesn't since some of the flags to go tool pack are different. We assume nobody is terribly bothered about this..."`
		GoRoot    string `help:"If set, will set the GOROOT environment variable appropriately during build actions."`
		TestTool  string `help:"Sets the location of the please_go_test tool that is used to template the test main for go_test rules."`
		GoPath    string `help:"If set, will set the GOPATH environment variable appropriately during build actions."`
		CgoCCTool string `help:"Sets the location of CC while building cgo_library and cgo_test rules. Defaults to gcc"`
	}
	Python struct {
		PipTool            string `help:"The tool that is invoked during pip_library rules. Defaults to, well, pip."`
		PipFlags           string
		PexTool            string  `help:"The tool that's invoked to build pexes. Defaults to please_pex in the install directory."`
		DefaultInterpreter string  `help:"The interpreter used for python_binary and python_test rules when none is specified on the rule itself. Defaults to python but you could of course set it to pypy."`
		ModuleDir          string  `help:"Defines a directory containing modules from which they can be imported at the top level.\nBy default this is empty but by convention we define our pip_library rules in third_party/python and set this appropriately. Hence any of those third-party libraries that try something like import six will have it work as they expect, even though it's actually in a different location within the .pex."`
		DefaultPipRepo     cli.URL `help:"Defines a location for a pip repo to download wheels from.\nBy default pip_library uses PyPI (although see below on that) but you may well want to use this define another location to upload your own wheels to.\nIs overridden by the repo argument to pip_library."`
		WheelRepo          cli.URL
		UsePyPI            bool `help:"Whether or not to use PyPI for pip_library rules or not. Defaults to true, if you disable this you will presumably want to set DefaultPipRepo to use one of your own.\nIs overridden by the use_pypi argument to pip_library."`
	}
	Java struct {
		JavacTool          string `help:"Defines the tool used for the Java compiler. Defaults to javac."`
		JavacWorker        string
		JarTool            string `help:"Defines the tool used to build a .jar. Defaults to jar."`
		JarCatTool         string `help:"Defines the tool used to concatenate .jar files which we use to build the output of java_binary and java_test.Defaults to jarcat in the Please install directory."`
		PleaseMavenTool    string `help:"Defines the tool used to fetch information from Maven in maven_jars rules.\nDefaults to please_maven in the Please install directory."`
		JUnitRunner        string `help:"Defines the .jar containing the JUnit runner. This is built into all java_test rules since it's necessary to make JUnit do anything useful.\nDefaults to junit_runner.jar in the Please install directory."`
		DefaultTestPackage string `help:"The Java classpath to search for functions annotated with @Test."`
		SourceLevel        string `help:"The default Java source level when compiling. Defaults to 8."`
		TargetLevel        string `help:"The default Java bytecode level to target. Defaults to 8."`
		JavacFlags         string
		JavacTestFlags     string
		DefaultMavenRepo   cli.URL
	}
	Cpp struct {
		CCTool             string `help:"The tool invoked to compile C code. Defaults to gcc but you might want to set it to clang, for example."`
		CppTool            string `help:"The tool invoked to compile C++ code. Defaults to g++ but you might want to set it to clang++, for example."`
		LdTool             string `help:"The tool invoked to link object files. Defaults to ld but you could also set it to gold, for example."`
		ArTool             string `help:"The tool invoked to archive static libraries. Defaults to ar."`
		AsmTool            string `help:"The tool invoked as an assembler. Currently only used on OSX for cc_embed_binary rules and so defaults to nasm."`
		LinkWithLdTool     bool   `help:"If true, instructs Please to use the tool set earlier in ldtool to link binaries instead of cctool.\nThis is an esoteric setting that most people don't want; a vanilla ld will not perform all steps necessary here (you'll get lots of missing symbol messages from having no libc etc). Generally best to leave this disabled."`
		DefaultOptCflags   string `help:"Compiler flags passed to all C rules during opt builds; these are typically pretty basic things like what language standard you want to target, warning flags, etc.\nDefaults to --std=c99 -O3 -DNDEBUG -Wall -Wextra -Werror"`
		DefaultDbgCflags   string `help:"Compiler rules passed to all C rules during dbg builds.\nDefaults to --std=c99 -g3 -DDEBUG -Wall -Wextra -Werror."`
		DefaultOptCppflags string `help:"Compiler flags passed to all C++ rules during opt builds; these are typically pretty basic things like what language standard you want to target, warning flags, etc.\nDefaults to --std=c++11 -O3 -DNDEBUG -Wall -Wextra -Werror"`
		DefaultDbgCppflags string `help:"Compiler rules passed to all C++ rules during dbg builds.\nDefaults to --std=c++11 -g3 -DDEBUG -Wall -Wextra -Werror."`
		DefaultLdflags     string `help:"Linker flags passed to all C++ rules.\nBy default this is empty."`
		DefaultNamespace   string `help:"Namespace passed to all cc_embed_binary rules when not overridden by the namespace argument to that rule.\nNot set by default, if you want to use those rules you'll need to set it or pass it explicitly to each one."`
		Coverage           bool   `help:"If true (the default), coverage will be available for C and C++ build rules.\nThis is still a little experimental but should work for GCC. Right now it does not work for Clang (it likely will in Clang 4.0 which will likely support --fprofile-dir) and so this can be useful to disable it.\nIt's also useful in some cases for CI systems etc if you'd prefer to avoid the overhead, since the tests have to be compiled with extra instrumentation and without optimisation."`
	}
	Proto struct {
		ProtocTool       string `help:"The binary invoked to compile .proto files. Defaults to protoc."`
		ProtocGoPlugin   string `help:"The binary passed to protoc as a plugin to generate Go code. Defaults to protoc-gen-go.\nWe've found this easier to manage with a go_get rule instead though, so you can also pass a build label here. See the Please repo for an example."`
		GrpcPythonPlugin string `help:"The plugin invoked to compile Python code for grpc_library.\nDefaults to protoc-gen-grpc-python."`
		GrpcJavaPlugin   string `help:"The plugin invoked to compile Java code for grpc_library.\nDefaults to protoc-gen-grpc-java."`
		GrpcCCPlugin     string
		Language         []string `help:"Sets the default set of languages that proto rules are built for.\nChosen from the set of {cc, java, go, py}.\nDefaults to all of them!"`
		PythonDep        string   `help:"An in-repo dependency that's applied to any Python targets built."`
		JavaDep          string   `help:"An in-repo dependency that's applied to any Java targets built."`
		GoDep            string   `help:"An in-repo dependency that's applied to any Go targets built."`
		JsDep            string
		PythonGrpcDep    string `help:"An in-repo dependency that's applied to any Python gRPC targets built."`
		JavaGrpcDep      string `help:"An in-repo dependency that's applied to any Java gRPC targets built."`
		GoGrpcDep        string `help:"An in-repo dependency that's applied to any Go gRPC targets built."`
		PythonPackage    string
	}
	Licences struct {
		Accept []string `help:"Licences that are accepted in this repository.\nWhen this is empty licences are ignored. As soon as it's set any licence detected or assigned must be accepted explicitly here.\nThere's no fuzzy matching, so some package managers (especially PyPI and Maven, but shockingly not npm which rather nicely uses SPDX) will generate a lot of slightly different spellings of the same thing, which will all have to be accepted here. We'd rather that than trying to 'cleverly' match them which might result in matching the wrong thing."`
		Reject []string `help:"Licences that are explicitly rejected in this repository.\nAn astute observer will notice that this is not very different to just not adding it to the accept section, but it does have the advantage of explicitly documenting things that the team aren't allowed to use."`
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
