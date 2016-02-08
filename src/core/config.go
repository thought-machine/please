// Utilities for reading the Please config files.

package core

import (
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"os"

	"gopkg.in/gcfg.v1"
)

// File name for the typical repo config - this is normally checked in
const ConfigFileName string = ".plzconfig"

// File name for the local repo config - this is not normally checked in and used to
// override settings on the local machine.
const LocalConfigFileName string = ".plzconfig.local"

// File name for the machine-level config - can use this to override things
// for a particular machine (eg. build machine with different caching behaviour).
const MachineConfigFileName = "/etc/plzconfig"

const PleaseVersion string = "<git>"

const TestContainerDocker = "docker"
const TestContainerNone = "none"

func readConfigFile(config *Configuration, filename string) error {
	if err := gcfg.ReadFileInto(config, filename); err != nil && os.IsNotExist(err) {
		return nil // It's not an error to not have the file at all.
	} else if err != nil {
		return err
	}
	// TODO(pebers): Use gcfg's types thingy to parse this once it's finalised.
	if config.Test.DefaultContainer != TestContainerNone && config.Test.DefaultContainer != TestContainerDocker {
		return fmt.Errorf("Unknown container type %s", config.Test.DefaultContainer)
	}
	return nil
}

// Reads a config file from the given locations, in order.
// Values are filled in by defaults initially and then overridden by each file in turn.
func ReadConfigFiles(filenames []string) (Configuration, error) {
	config := DefaultConfiguration()
	for _, filename := range filenames {
		if err := readConfigFile(&config, filename); err != nil {
			return config, err
		}
	}
	// Set default values for slices. These add rather than overwriting so we can't set
	// them upfront as we would with other config values.
	setDefault(&config.Please.BuildFileName, []string{"BUILD"})
	setDefault(&config.Build.Path, []string{"/usr/local/bin", "/usr/bin", "/bin"})
	setDefault(&config.Please.PyPyLocation, []string{"/usr/lib/libpypy-c.so", "/usr/local/lib/libpypy-c.dylib"})
	setDefault(&config.Cover.FileExtension, []string{".go", ".py", ".java", ".js", ".cc", ".h", ".c"})
	setDefault(&config.Cover.ExcludeExtension, []string{".pb.go", "_pb2.py", ".pb.cc", ".pb.h", "_test.py"})
	setDefault(&config.Proto.Language, []string{"cc", "py", "java", "go"})

	return config, nil
}

// setDefault sets a slice of strings in the config if the set one is empty.
func setDefault(conf *[]string, def []string) {
	if len(*conf) == 0 {
		*conf = def
	}
}

func DefaultConfiguration() Configuration {
	config := Configuration{}
	config.Please.Version = PleaseVersion
	config.Please.Location = "/opt/please"
	config.Please.SelfUpdate = true
	config.Please.DownloadLocation = "https://s3-eu-west-1.amazonaws.com/please-build"
	config.Please.Lang = "en_GB.UTF-8" // Not the language of the UI, the language passed to rules.
	config.Please.Nonce = "1402"       // Arbitrary nonce to invalidate config when needed.
	config.Build.Timeout = 600         // Ten minutes
	config.Build.Config = "opt"        // Optimised builds by default
	config.Build.DefaultConfig = "opt" // Optimised builds as a fallback on any target that doesn't have a matching one set
	config.Cache.HttpTimeout = 5       // Five seconds
	config.Cache.RpcTimeout = 5        // Five seconds
	config.Cache.DirCacheCleaner = "/opt/please/cache_cleaner"
	config.Cache.DirCacheHighWaterMark = "10G"
	config.Cache.DirCacheLowWaterMark = "8G"
	config.Test.Timeout = 600
	config.Test.DefaultContainer = TestContainerDocker
	config.Docker.DefaultImage = "ubuntu:trusty"
	config.Docker.AllowLocalFallback = true
	config.Docker.Timeout = 1200      // Twenty minutes
	config.Docker.ResultsTimeout = 20 // Twenty seconds
	config.Docker.RemoveTimeout = 20  // Twenty seconds
	config.Go.Version = "1.5.1"
	config.Go.TestTool = "/opt/please/please_go_test"
	config.Python.PipTool = "pip"
	config.Python.PexTool = "/opt/please/please_pex"
	config.Python.DefaultInterpreter = "/usr/bin/python"
	config.Python.UsePyPI = true
	config.Java.JavacTool = "javac"
	config.Java.JarTool = "jar"
	config.Java.JarCatTool = "/opt/please/jarcat"
	config.Java.PleaseMavenTool = "/opt/please/please_maven"
	config.Java.JUnitRunner = "/opt/please/junit_runner.jar"
	config.Java.DefaultTestPackage = ""
	config.Java.SourceLevel = "8"
	config.Java.TargetLevel = "8"
	config.Cpp.CCTool = "g++"
	config.Cpp.LdTool = "ld"
	config.Cpp.DefaultOptCflags = "--std=c++11 -O2 -DNDEBUG -Wall -Wextra -Werror"
	config.Cpp.DefaultDbgCflags = "--std=c++11 -g3 -DDEBUG -Wall -Wextra -Werror"
	config.Proto.ProtocTool = "protoc"
	config.Proto.ProtocGoPlugin = "`which protoc-gen-go`" // These seem to need absolute paths
	config.Proto.GrpcPythonPlugin = "`which protoc-gen-grpc-python`"
	config.Proto.GrpcJavaPlugin = "`which protoc-gen-grpc-java`"
	config.Proto.ProtocVersion = ""
	config.Proto.PythonDep = "//third_party/python:protobuf"
	config.Proto.JavaDep = "//third_party/java:protobuf"
	config.Proto.GoDep = "//third_party/go:protobuf"
	config.Proto.GrpcVersion = ""
	config.Proto.PythonGrpcDep = "//third_party/python:grpc"
	config.Proto.JavaGrpcDep = "//third_party/java:grpc-all"
	config.Proto.GoGrpcDep = "//third_party/go:grpc"
	return config
}

type Configuration struct {
	Please struct {
		Version          string
		Location         string
		SelfUpdate       bool
		DownloadLocation string
		BuildFileName    []string
		Lang             string
		PyPyLocation     []string
		Nonce            string
	}
	Build struct {
		Timeout       int
		Path          []string
		Config        string
		DefaultConfig string
	}
	Cache struct {
		Dir                   string
		DirCacheCleaner       string
		DirCacheHighWaterMark string
		DirCacheLowWaterMark  string
		HttpUrl               string
		HttpWriteable         bool
		HttpTimeout           int
		RpcUrl                string
		RpcWriteable          bool
		RpcTimeout            int
	}
	Test struct {
		Timeout          int
		DefaultContainer string
	}
	Cover struct {
		FileExtension    []string
		ExcludeExtension []string
	}
	Docker struct {
		DefaultImage       string
		AllowLocalFallback bool
		Timeout            int
		ResultsTimeout     int
		RemoveTimeout      int
		RunArgs            []string
	}
	Go struct {
		Version  string
		TestTool string
	}
	Python struct {
		PipTool            string
		PexTool            string
		DefaultInterpreter string
		ModuleDir          string
		DefaultPipRepo     string
		UsePyPI            bool
	}
	Java struct {
		JavacTool          string
		JarTool            string
		JarCatTool         string
		PleaseMavenTool    string
		JUnitRunner        string
		DefaultTestPackage string
		SourceLevel        string
		TargetLevel        string
	}
	Cpp struct {
		CCTool           string
		LdTool           string
		DefaultOptCflags string
		DefaultDbgCflags string
		DefaultLdflags   string
		DefaultNamespace string
	}
	Proto struct {
		ProtocTool       string
		ProtocGoPlugin   string
		GrpcPythonPlugin string
		GrpcJavaPlugin   string
		Language         []string
		ProtocVersion    string
		PythonDep        string
		JavaDep          string
		GoDep            string
		GrpcVersion      string
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
	h.Write([]byte(config.Go.Version)) // Sucks a bit not to invalidate just Go rules but it's hard to detect.
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
