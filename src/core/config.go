// Utilities for reading the Please config files.

package core

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/google/shlex"
	"github.com/please-build/gcfg"
	gcfgtypes "github.com/please-build/gcfg/types"
	"github.com/thought-machine/go-flags"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/fs"
)

// OsArch is the os/arch pair, like linux_amd64 etc.
const OsArch = runtime.GOOS + "_" + runtime.GOARCH

// ConfigName is the base name for config files.
const ConfigName string = "plzconfig"

// ConfigFileName is the file name for the typical repo config - this is normally checked in
const ConfigFileName string = ".plzconfig"

// ArchConfigFileName is the architecture-specific config file which overrides the repo one.
// Also normally checked in if needed.
const ArchConfigFileName string = ".plzconfig_" + OsArch

// LocalConfigFileName is the file name for the local repo config - this is not normally checked
// in and used to override settings on the local machine.
const LocalConfigFileName string = ".plzconfig.local"

// MachineConfigFileName is the file name for the machine-level config - can use this to override
// things for a particular machine (eg. build machine with different caching behaviour).
const MachineConfigFileName = "/etc/please/plzconfig"

// UserConfigFileName is the file name for user-specific config (for all their repos).
const UserConfigFileName = "~/.config/please/plzconfig"

// DefaultPleaseLocation is the default location where Please is installed.
const DefaultPleaseLocation = "~/.please"

// DefaultPath is the default location please looks for programs in
var DefaultPath = []string{"/usr/local/bin", "/usr/bin", "/bin"}

// readConfigFileOnly reads a single config file into the config struct
func readConfigFileOnly(config *Configuration, filename string) error {
	log.Debug("Attempting to read config from %s...", filename)
	if err := gcfg.ReadFileInto(config, filename); err != nil && os.IsNotExist(err) {
		return nil // It's not an error to not have the file at all.
	} else if gcfg.FatalOnly(err) != nil {
		return err
	} else if err != nil {
		log.Warning("Error in config file: %s", err)
	} else {
		log.Debug("Read config from %s", filename)
	}
	return nil
}

// readConfigFile reads a single config file into the config struct taking into account
// some context like subrepos and plugins.
func readConfigFile(config *Configuration, filename string, subrepo bool) error {
	if err := readConfigFileOnly(config, filename); err != nil {
		return err
	}

	if subrepo {
		checkPluginVersionRequirements(config)
	}
	normalisePluginConfigKeys(config)

	return nil
}

func checkPluginVersionRequirements(config *Configuration) {
	if config.PluginDefinition.Name != "" {
		currentPlzVersion := *semver.New(PleaseVersion)
		// Get plugin config version requirement which may or may not exist
		pluginVerReq := config.Please.Version.Version

		if currentPlzVersion.LessThan(pluginVerReq) {
			log.Warningf("Plugin \"%v\" requires Please version %v", config.PluginDefinition.Name, pluginVerReq)
		}
	}
}

// ReadDefaultConfigFiles reads all the config files from the default locations and
// merges them into a config object.
// The repo root must have already have been set before calling this.
func ReadDefaultConfigFiles(profiles []ConfigProfile) (*Configuration, error) {
	s := make([]string, len(profiles))
	for i, p := range profiles {
		s[i] = string(p)
	}
	return ReadConfigFiles(defaultConfigFiles(), s)
}

// ReadDefaultGlobalConfigFilesOnly reads all the default global config files and
// merges them into a config object.
func ReadDefaultGlobalConfigFilesOnly(config *Configuration) error {
	return ReadConfigFilesOnly(config, defaultGlobalConfigFiles(), nil)
}

// ReadDefaultConfigFilesOnly reads all the default config files and
// merges them into a config object.
func ReadDefaultConfigFilesOnly(config *Configuration, profiles []ConfigProfile) error {
	s := make([]string, len(profiles))
	for i, p := range profiles {
		s[i] = string(p)
	}
	return ReadConfigFilesOnly(config, defaultConfigFiles(), s)
}

// defaultGlobalConfigFiles returns the set of global default config file names.
func defaultGlobalConfigFiles() []string {
	configFiles := []string{
		MachineConfigFileName,
	}

	if xdgConfigDirs := os.Getenv("XDG_CONFIG_DIRS"); xdgConfigDirs != "" {
		for _, p := range strings.Split(xdgConfigDirs, ":") {
			if !strings.HasPrefix(p, "/") {
				continue
			}

			configFiles = append(configFiles, filepath.Join(p, ConfigName))
		}
	}

	// Note: according to the XDG Base Directory Specification,
	// this path should only be checked if XDG_CONFIG_HOME env var is not set,
	// but it should be kept here for backward compatibility purposes.
	configFiles = append(configFiles, fs.ExpandHomePath(UserConfigFileName))

	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" && strings.HasPrefix(xdgConfigHome, "/") {
		configFiles = append(configFiles, filepath.Join(xdgConfigHome, ConfigName))
	}

	return configFiles
}

// defaultConfigFiles returns the set of default config file names.
func defaultConfigFiles() []string {
	return append(
		defaultGlobalConfigFiles(), filepath.Join(RepoRoot, ConfigFileName), filepath.Join(RepoRoot, ArchConfigFileName), filepath.Join(RepoRoot, LocalConfigFileName),
	)
}

// ReadConfigFilesOnly reads all the config locations, in order, and merges them into a config object.
func ReadConfigFilesOnly(config *Configuration, filenames []string, profiles []string) error {
	for _, filename := range filenames {
		if err := readConfigFileOnly(config, filename); err != nil {
			return err
		}
		for _, profile := range profiles {
			if err := readConfigFileOnly(config, filename+"."+profile); err != nil {
				return err
			}
		}
	}
	return nil
}

// ReadConfigFiles reads all the config locations, in order, and merges them into a config object.
// Values are filled in by defaults initially and then overridden by each file in turn.
func ReadConfigFiles(filenames []string, profiles []string) (*Configuration, error) {
	config := DefaultConfiguration()
	for _, filename := range filenames {
		if err := readConfigFile(config, filename, false); err != nil {
			return config, err
		}
		for _, profile := range profiles {
			if err := readConfigFile(config, filename+"."+profile, false); err != nil {
				return config, err
			}
		}
	}

	// Set default values for slices. These add rather than overwriting so we can't set
	// them upfront as we would with other config values.
	setDefault(&config.Please.PluginRepo,
		"https://github.com/{owner}/{plugin}/archive/{revision}.zip",
		"https://github.com/{owner}/{plugin}-rules/archive/{revision}.zip",
	)
	if usingBazelWorkspace {
		setDefault(&config.Parse.BuildFileName, "BUILD.bazel", "BUILD", "BUILD.plz")
	} else {
		setDefault(&config.Parse.BuildFileName, "BUILD", "BUILD.plz")
	}
	setBuildPath(&config.Build.Path, config.Build.PassEnv, config.Build.PassUnsafeEnv)
	setDefault(&config.Build.HashCheckers, "sha1", "sha256", "blake3")
	setDefault(&config.Build.PassUnsafeEnv)
	setDefault(&config.Build.PassEnv)
	setDefault(&config.Cover.FileExtension, ".go", ".py", ".java", ".tsx", ".ts", ".js", ".cc", ".h", ".c")
	setDefault(&config.Cover.ExcludeExtension, ".pb.go", "_pb2.py", ".spec.tsx", ".spec.ts", ".spec.js", ".pb.cc", ".pb.h", "_test.py", "_test.go", "_pb.go", "_bindata.go", "_test_main.cc")
	setDefault(&config.Proto.Language, "cc", "py", "java", "go", "js")
	setDefault(&config.Parse.BuildDefsDir, "build_defs")

	if config.Go.GoRoot != "" {
		config.Go.GoTool = filepath.Join(config.Go.GoRoot, "bin", "go")
	}

	// Default values for these guys depend on config.Java.JavaHome if that's been set.
	if config.Java.JavaHome != "" {
		defaultPathIfExists(&config.Java.JlinkTool, config.Java.JavaHome, "bin/jlink")
	}

	if config.Colours == nil {
		config.Colours = map[string]string{
			"py":   "${GREEN}",
			"java": "${RED}",
			"go":   "${YELLOW}",
			"js":   "${BLUE}",
		}
	} else {
		// You are allowed to just write "yellow" but we map that to a pseudo-variable thing.
		for k, v := range config.Colours {
			if v[0] != '$' {
				config.Colours[k] = "${" + strings.ToUpper(v) + "}"
			}
		}
	}

	// In a few versions we will deprecate Cpp.Coverage completely in favour of this more generic scheme.
	if !config.Cpp.Coverage {
		config.Test.DisableCoverage = append(config.Test.DisableCoverage, "cc")
	}

	if len(config.Size) == 0 {
		config.Size = map[string]*Size{
			"small": {
				Timeout:     cli.Duration(1 * time.Minute),
				TimeoutName: "short",
			},
			"medium": {
				Timeout:     cli.Duration(5 * time.Minute),
				TimeoutName: "moderate",
			},
			"large": {
				Timeout:     cli.Duration(15 * time.Minute),
				TimeoutName: "long",
			},
			"enormous": {
				TimeoutName: "eternal",
			},
		}
	}
	// Dump the timeout names back in so we can look them up later
	for _, size := range config.Size {
		if size.TimeoutName != "" {
			config.Size[size.TimeoutName] = size
		}
	}

	// Resolve the full path to its location.
	config.EnsurePleaseLocation()

	// If the HTTP proxy config is set and there is no env var overriding it, set it now
	// so various other libraries will honour it.
	if config.Build.HTTPProxy != "" {
		os.Setenv("HTTP_PROXY", config.Build.HTTPProxy.String())
	}

	// Deal with the various sandbox settings that are moving.
	if config.Build.Sandbox {
		log.Warning("build.sandbox in config is deprecated, use sandbox.build instead")
		config.Sandbox.Build = true
	}
	if config.Test.Sandbox {
		log.Warning("test.sandbox in config is deprecated, use sandbox.test instead")
		config.Sandbox.Test = true
	}
	if config.Build.PleaseSandboxTool != "" {
		log.Warning("build.pleasesandboxtool in config is deprecated, use sandbox.tool instead")
		config.Sandbox.Tool = config.Build.PleaseSandboxTool
	}

	// We can only verify options by reflection (we need struct tags) so run them quickly through this.
	return config, config.ApplyOverrides(map[string]string{
		"build.hashfunction": config.Build.HashFunction,
		"build.hashcheckers": strings.Join(config.Build.HashCheckers, ","),
	})
}

// normalisePluginConfigKeys converts all config for plugins to lower case
func normalisePluginConfigKeys(config *Configuration) {
	for _, plugin := range config.Plugin {
		newExtraValues := make(map[string][]string, len(plugin.ExtraValues))
		for k, v := range plugin.ExtraValues {
			_, ok := newExtraValues[strings.ToLower(k)]
			if k == strings.ToLower(k) && ok {
				// We have to handle overriding plugin config with .plzconfig files of higher precedence e.g. profiles
				//
				// When we meet this condition that means that the non-normalised config from the new .plzconfig file
				// has been loaded, so we don't need to do anything. If that config used the already normalized form of
				// the config key then it would've been overridden in the gcfg library, so we don't need to handle that
				// case.
				continue
			}
			newExtraValues[strings.ToLower(k)] = v
		}
		plugin.ExtraValues = newExtraValues
	}
}

// setDefault sets a slice of strings in the config if the set one is empty.
func setDefault(conf *[]string, def ...string) {
	if len(*conf) == 0 {
		*conf = def
	}
}

// setDefault checks if "PATH" is in passEnv, if it is set config.build.Path to use the environment variable.
func setBuildPath(conf *[]string, passEnv []string, passUnsafeEnv []string) {
	pathVal := DefaultPath
	for _, i := range passUnsafeEnv {
		if i == "PATH" {
			pathVal = strings.Split(os.Getenv("PATH"), ":")
		}
	}
	for _, i := range passEnv {
		if i == "PATH" {
			pathVal = strings.Split(os.Getenv("PATH"), ":")
		}
	}
	setDefault(conf, pathVal...)
}

// defaultPathIfExists sets a variable to a location in a directory if it's not already set and if the location exists.
func defaultPathIfExists(conf *string, dir, file string) {
	if *conf == "" {
		location := filepath.Join(dir, file)
		// check that the location is valid
		if _, err := os.Stat(location); err == nil {
			*conf = location
		}
	}
}

// DefaultConfiguration returns the default configuration object with no overrides.
// N.B. Slice fields are not populated by this (since it interferes with reading them)
func DefaultConfiguration() *Configuration {
	config := Configuration{buildEnvStored: &storedBuildEnv{}}
	config.Please.SelfUpdate = true
	config.Please.Autoclean = true
	config.Please.DownloadLocation = "https://get.please.build"
	config.Please.NumOldVersions = 10
	config.Please.NumThreads = runtime.NumCPU() + 2
	config.Parse.NumThreads = config.Please.NumThreads
	config.Parse.GitFunctions = true
	config.Build.Arch = cli.NewArch(runtime.GOOS, runtime.GOARCH)
	config.Build.Lang = "en_GB.UTF-8" // Not the language of the UI, the language passed to rules.
	config.Build.Nonce = "1402"       // Arbitrary nonce to invalidate config when needed.
	config.Build.Timeout = cli.Duration(10 * time.Minute)
	config.Build.Config = "opt"         // Optimised builds by default
	config.Build.FallbackConfig = "opt" // Optimised builds as a fallback on any target that doesn't have a matching one set
	config.Build.Xattrs = true
	config.Build.HashFunction = "sha256"
	config.BuildConfig = map[string]string{}
	config.BuildEnv = map[string]string{}
	config.Cache.HTTPWriteable = true
	config.Cache.HTTPTimeout = cli.Duration(25 * time.Second)
	config.Cache.HTTPConcurrentRequestLimit = 20
	config.Cache.HTTPRetry = 4
	if dir, err := os.UserCacheDir(); err == nil {
		config.Cache.Dir = filepath.Join(dir, "please")
	}
	config.Cache.DirCacheHighWaterMark = 10 * cli.GiByte
	config.Cache.DirCacheLowWaterMark = 8 * cli.GiByte
	config.Cache.DirClean = true
	config.Cache.Workers = runtime.NumCPU() + 2 // Mirrors the number of workers in please.go.
	config.Test.Timeout = cli.Duration(10 * time.Minute)
	config.Display.SystemStats = true
	config.Display.MaxWorkers = 40
	config.Display.ColourScheme = "dark"
	config.Remote.NumExecutors = 20 // kind of arbitrary
	config.Remote.Secure = true
	config.Remote.VerifyOutputs = true
	config.Remote.UploadDirs = true
	config.Remote.CacheDuration = cli.Duration(10000 * 24 * time.Hour) // Effectively forever.
	config.Go.GoTool = "go"
	config.Go.CgoCCTool = "gcc"
	config.Go.DelveTool = "dlv"
	config.Python.DefaultInterpreter = "python3"
	config.Python.DisableVendorFlags = false
	config.Python.TestRunner = "unittest"
	config.Python.TestRunnerBootstrap = ""
	config.Python.Debugger = "pdb"
	config.Python.UsePyPI = true
	config.Python.InterpreterOptions = ""
	config.Python.PipFlags = ""
	config.Java.DefaultTestPackage = ""
	config.Java.SourceLevel = "8"
	config.Java.TargetLevel = "8"
	config.Java.ReleaseLevel = ""
	config.Java.DefaultMavenRepo = []cli.URL{"https://repo1.maven.org/maven2", "https://jcenter.bintray.com/"}
	config.Java.JavacFlags = "-Werror -Xlint:-options" // bootstrap class path warnings are pervasive without this.
	config.Java.JlinkTool = "jlink"
	config.Java.JavaHome = ""
	config.Cpp.CCTool = "gcc"
	config.Cpp.CppTool = "g++"
	config.Cpp.LdTool = "ld"
	config.Cpp.ArTool = "ar"
	config.Cpp.DefaultOptCflags = "--std=c99 -O3 -pipe -DNDEBUG -Wall -Werror"
	config.Cpp.DefaultDbgCflags = "--std=c99 -g3 -pipe -DDEBUG -Wall -Werror"
	config.Cpp.DefaultOptCppflags = "--std=c++11 -O3 -pipe -DNDEBUG -Wall -Werror"
	config.Cpp.DefaultDbgCppflags = "--std=c++11 -g3 -pipe -DDEBUG -Wall -Werror"
	config.Cpp.Coverage = true
	config.Cpp.ClangModules = true
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
	config.Remote.Timeout = cli.Duration(2 * time.Minute)
	config.Bazel.Compatibility = usingBazelWorkspace

	config.Sandbox.Tool = "please_sandbox"
	// Please tools
	config.Go.FilterTool = "/////_please:please_go_filter"
	config.Go.PleaseGoTool = "/////_please:please_go"
	config.Go.EmbedTool = "/////_please:please_go_embed"
	config.Python.PexTool = "/////_please:please_pex"
	config.Java.JavacWorker = "/////_please:javac_worker"
	config.Java.JarCatTool = "/////_please:arcat"
	config.Java.JUnitRunner = "/////_please:junit_runner"

	config.Metrics.Timeout = cli.Duration(2 * time.Second)

	return &config
}

// A Configuration contains all the settings that can be configured about Please.
// This is parsed from .plzconfig etc; we also auto-generate help messages from its tags.
type Configuration struct {
	Please struct {
		Version          cli.Version `help:"Defines the version of plz that this repo is supposed to use currently. If it's not present or the version matches the currently running version no special action is taken; otherwise if SelfUpdate is set Please will attempt to download an appropriate version, otherwise it will issue a warning and continue.\n\nNote that if this is not set, you can run plz update to update to the latest version available on the server." var:"PLZ_VERSION"`
		ToolsURL         cli.URL     `help:"The URL download the Please tools from. Defaults to download the tools from the current Please versions github releases page."`
		VersionChecksum  []string    `help:"Defines a hex-encoded sha256 checksum that the downloaded version must match. Can be specified multiple times to support different architectures." example:"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"`
		Location         string      `help:"Defines the directory Please is installed into.\nDefaults to ~/.please but you might want it to be somewhere else if you're installing via another method (e.g. the debs and install script still use /opt/please)."`
		SelfUpdate       bool        `help:"Sets whether plz will attempt to update itself when the version set in the config file is different."`
		DownloadLocation cli.URL     `help:"Defines the location to download Please from when self-updating. Defaults to the Please web server, but you can point it to some location of your own if you prefer to keep traffic within your network or use home-grown versions."`
		NumOldVersions   int         `help:"Number of old versions to keep from autoupdates."`
		Autoclean        bool        `help:"Automatically clean stale versions without prompting"`
		NumThreads       int         `help:"Number of parallel build operations to run.\nIs overridden by the equivalent command-line flag, if that's passed." example:"6"`
		Motd             []string    `help:"Message of the day; is displayed once at the top during builds. If multiple are given, one is randomly chosen."`
		DefaultRepo      string      `help:"Location of the default repository; this is used if plz is invoked when not inside a repo, it changes to that directory then does its thing."`
		PluginRepo       []string    `help:"A list of template URLS used to download plugins from. The download should be an archive e.g. .tar.gz, or .zip. Templatized variables should be surrounded in curly braces, and the available options are: owner, revision and plugin. Defaults to github and gitlab." example:"https://gitlab.you.org/{owner}/{plugin}/-/archive/{revision}/{plugin}-{revision}.zip" var:"PLUGIN_REPOS"`
	} `help:"The [please] section in the config contains non-language-specific settings defining how Please should operate."`
	Parse struct {
		ExperimentalDir    []string     `help:"Directory containing experimental code. This is subject to some extra restrictions:\n - Code in the experimental dir can override normal visibility constraints\n - Code outside the experimental dir can never depend on code inside it\n - Tests are excluded from general detection." example:"experimental"`
		BuildFileName      []string     `help:"Sets the names that Please uses instead of BUILD for its build files.\nFor clarity the documentation refers to them simply as BUILD files but you could reconfigure them here to be something else.\nOne case this can be particularly useful is in cases where you have a subdirectory named build on a case-insensitive file system like HFS+." var:"BUILD_FILE_NAMES"`
		BlacklistDirs      []string     `help:"Directories to blacklist when recursively searching for BUILD files (e.g. when using plz build ... or similar).\nThis is generally useful when you have large directories within your repo that don't need to be searched, especially things like node_modules that have come from external package managers."`
		PreloadBuildDefs   []string     `help:"Files to preload by the parser before loading any BUILD files.\nSince this is done before the first package is parsed they must be files in the repository, they cannot be subinclude() paths. Use Init instead." example:"build_defs/go_bindata.build_defs"`
		PreloadSubincludes []BuildLabel `help:"Subinclude targets to preload by the parser before loading any BUILD files.\nSubincludes can be slow so it's recommended to use PreloadBuildDefs where possible." example:"///pleasings//python:requirements"`
		BuildDefsDir       []string     `help:"Directory to look in when prompted for help topics that aren't known internally." example:"build_defs"`
		NumThreads         int          `help:"Number of parallel parse operations to run.\nIs overridden by the --num_threads command line flag." example:"6"`
		GitFunctions       bool         `help:"Activates built-in functions git_branch, git_commit, git_show and git_state. If disabled they will not be usable at parse time."`
	} `help:"The [parse] section in the config contains settings specific to parsing files."`
	Display struct {
		UpdateTitle  bool   `help:"Updates the title bar of the shell window Please is running in as the build progresses. This isn't on by default because not everyone's shell is configured to reset it again after and we don't want to alter it forever."`
		SystemStats  bool   `help:"Whether or not to show basic system resource usage in the interactive display. Has no effect without that configured."`
		MaxWorkers   int    `help:"Maximum number of worker rows to display at any one time."`
		ColourScheme string `help:"Shell colour scheme mode, dark or light. Defaults to dark"`
	} `help:"Please has an animated display mode which shows the currently building targets.\nBy default it will autodetect whether it is using an interactive TTY session and choose whether to use it or not, although you can force it on or off via flags.\n\nThe display is heavily inspired by Buck's SuperConsole."`
	Colours map[string]string `help:"Colour code overrides for the targets in interactive output. These colours are map labels on targets to colours e.g. go -> ${YELLOW}."`
	Build   struct {
		Arch                 cli.Arch     `help:"The target architecture to compile for. Defaults to the host architecture."`
		Timeout              cli.Duration `help:"Default timeout for build actions. Default is ten minutes."`
		Path                 []string     `help:"The PATH variable that will be passed to the build processes.\nDefaults to /usr/local/bin:/usr/bin:/bin but of course can be modified if you need to get binaries from other locations." example:"/usr/local/bin:/usr/bin:/bin"`
		Config               string       `help:"The build config to use when one is not chosen on the command line. Defaults to opt." example:"opt | dbg"`
		FallbackConfig       string       `help:"The build config to use when one is chosen and a required target does not have one by the same name. Also defaults to opt." example:"opt | dbg"`
		Lang                 string       `help:"Sets the language passed to build rules when building. This can be important for some tools (although hopefully not many) - we've mostly observed it with Sass."`
		Sandbox              bool         `help:"Deprecated, use sandbox.build instead."`
		Xattrs               bool         `help:"True (the default) to attempt to use xattrs to record file metadata. If false Please will fall back to using additional files where needed, which is more compatible but has slightly worse performance."`
		PleaseSandboxTool    string       `help:"Deprecated, use sandbox.tool instead."`
		Nonce                string       `help:"This is an arbitrary string that is added to the hash of every build target. It provides a way to force a rebuild of everything when it's changed.\nWe will bump the default of this whenever we think it's required - although it's been a pretty long time now and we hope that'll continue."`
		PassEnv              []string     `help:"A list of environment variables to pass from the current environment to build rules. For example\n\nPassEnv = HTTP_PROXY\n\nwould copy your HTTP_PROXY environment variable to the build env for any rules."`
		PassUnsafeEnv        []string     `help:"Similar to PassEnv, a list of environment variables to pass from the current environment to build rules. Unlike PassEnv, the environment variable values are not used when calculating build target hashes."`
		HTTPProxy            cli.URL      `help:"A URL to use as a proxy server for downloads. Only applies to internal ones - e.g. self-updates or remote_file rules."`
		HashCheckers         []string     `help:"Set of hash algos supported by the 'hashes' argument on build rules. Defaults to: sha1,sha256,blake3." options:"sha1,sha256,blake3,xxhash,crc32,crc64"`
		HashFunction         string       `help:"The hash function to use internally for build actions." options:"sha1,sha256,blake3,xxhash,crc32,crc64"`
		ExitOnError          bool         `help:"True to have build actions automatically fail on error (essentially passing -e to the shell they run in)." var:"EXIT_ON_ERROR"`
		DownloadLinkable     bool         `help:"True to download targets on remote that have links defined."`
		LinkGeneratedSources string       `help:"If set, supported build definitions will link generated sources back into the source tree. The list of generated files can be generated for the .gitignore through 'plz query print --label gitignore: //...'. The available options are: 'hard' (hardlinks), 'soft' (symlinks), 'true' (symlinks) and 'false' (default)"`
		UpdateGitignore      bool         `help:"Whether to automatically update the nearest gitignore with generated sources"`
	} `help:"A config section describing general settings related to building targets in Please.\nSince Please is by nature about building things, this only has the most generic properties; most of the more esoteric properties are configured in their own sections."`
	BuildConfig map[string]string `help:"A section of arbitrary key-value properties that are made available in the BUILD language. These are often useful for writing custom rules that need some configurable property.\n\n[buildconfig]\nandroid-tools-version = 23.0.2\n\nFor example, the above can be accessed as CONFIG.ANDROID_TOOLS_VERSION."`
	BuildEnv    map[string]string `help:"A set of extra environment variables to define for build rules. For example:\n\n[buildenv]\nsecret-passphrase = 12345\n\nThis would become SECRET_PASSPHRASE for any rules. These can be useful for passing secrets into custom rules; any variables containing SECRET or PASSWORD won't be logged.\n\nIt's also useful if you'd like internal tools to honour some external variable."`
	Cache       struct {
		Workers                    int          `help:"Number of workers for uploading artifacts to remote caches, which is done asynchronously."`
		Dir                        string       `help:"Sets the directory to use for the dir cache.\nThe default is 'please' under the user's cache dir (i.e. ~/.cache/please, ~/Library/Caches/please, etc), if set to the empty string the dir cache will be disabled." example:".plz-cache"`
		DirCacheHighWaterMark      cli.ByteSize `help:"Starts cleaning the directory cache when it is over this number of bytes.\nCan also be given with human-readable suffixes like 10G, 200MB etc."`
		DirCacheLowWaterMark       cli.ByteSize `help:"When cleaning the directory cache, it's reduced to at most this size."`
		DirClean                   bool         `help:"Controls whether entries in the dir cache are cleaned or not. If disabled the cache will only grow."`
		DirCompress                bool         `help:"Compresses stored artifacts in the dir cache. They are slower to store & retrieve but more compact."`
		HTTPURL                    cli.URL      `help:"Base URL of the HTTP cache.\nNot set to anything by default which means the cache will be disabled."`
		HTTPWriteable              bool         `help:"If True this plz instance will write content back to the HTTP cache.\nBy default it runs in read-only mode."`
		HTTPTimeout                cli.Duration `help:"Timeout for operations contacting the HTTP cache, in seconds."`
		HTTPConcurrentRequestLimit int          `help:"The maximum amount of concurrent requests that can be open. Default 20."`
		HTTPRetry                  int          `help:"The maximum number of retries before a request will give up, if a request is retryable"`
		StoreCommand               string       `help:"Use a custom command to store cache entries."`
		RetrieveCommand            string       `help:"Use a custom command to retrieve cache entries."`
	} `help:"Please has several built-in caches that can be configured in its config file.\n\nThe simplest one is the directory cache which by default is written into the .plz-cache directory. This allows for fast retrieval of code that has been built before (for example, when swapping Git branches).\n\nThere is also a remote RPC cache which allows using a centralised server to store artifacts. A typical pattern here is to have your CI system write artifacts into it and give developers read-only access so they can reuse its work.\n\nFinally there's a HTTP cache which is very similar, but a little obsolete now since the RPC cache outperforms it and has some extra features. Otherwise the two have similar semantics and share quite a bit of implementation.\n\nPlease has server implementations for both the RPC and HTTP caches."`
	Test struct {
		Timeout                  cli.Duration `help:"Default timeout applied to all tests. Can be overridden on a per-rule basis."`
		Sandbox                  bool         `help:"Deprecated, use sandbox.test instead."`
		DisableCoverage          []string     `help:"Disables coverage for tests that have any of these labels spcified."`
		Upload                   cli.URL      `help:"URL to upload test results to (in XML format)"`
		UploadGzipped            bool         `help:"True to upload the test results gzipped."`
		StoreTestOutputOnSuccess bool         `help:"True to store stdout and stderr in the test results for successful tests."`
	} `help:"A config section describing settings related to testing in general."`
	Sandbox struct {
		Tool               string       `help:"The location of the tool to use for sandboxing. This can assume it is being run in a new network, user, and mount namespace on linux. If not set, Please will use 'plz sandbox'."`
		Dir                []string     `help:"Directories to hide within the sandbox"`
		Namespace          string       `help:"Set to 'always', to namespace all actions. Set to 'sandbox' to namespace only when sandboxing the build action. Defaults to 'never', under the assumption the sandbox tool will handle its own namespacing. If set, user namespacing will be enabled for all rules. Mount and network will only be enabled if the rule is to be sandboxed."`
		Build              bool         `help:"True to sandbox individual build actions, which isolates them from network access and some aspects of the filesystem. Currently only works on Linux." var:"BUILD_SANDBOX"`
		Test               bool         `help:"True to sandbox individual tests, which isolates them from network access, IPC and some aspects of the filesystem. Currently only works on Linux." var:"TEST_SANDBOX"`
		ExcludeableTargets []BuildLabel `help:"If set, only targets that match these wildcards will be allowed to opt out of the sandbox"`
	} `help:"A config section describing settings relating to sandboxing of build actions."`
	Remote struct {
		URL           string       `help:"URL for the remote server."`
		CASURL        string       `help:"URL for the CAS service, if it is different to the main one."`
		AssetURL      string       `help:"URL for the remote asset server, if it is different to the main one."`
		NumExecutors  int          `help:"Maximum number of remote executors to use simultaneously."`
		Instance      string       `help:"Remote instance name to request; depending on the server this may be required."`
		Name          string       `help:"A name for this worker instance. This is attached to artifacts uploaded to remote storage." example:"agent-001"`
		DisplayURL    string       `help:"A URL to browse the remote server with (e.g. using buildbarn-browser). Only used when printing hashes."`
		TokenFile     string       `help:"A file containing a token that is attached to outgoing RPCs to authenticate them. This is somewhat bespoke; we are still investigating further options for authentication."`
		Timeout       cli.Duration `help:"Timeout for connections made to the remote server."`
		Secure        bool         `help:"Whether to use TLS for communication or not."`
		VerifyOutputs bool         `help:"Whether to verify all outputs are present after a cached remote execution action. Depending on your server implementation, you may require this to ensure files are really present."`
		UploadDirs    bool         `help:"Uploads individual directory blobs after build actions. This might not be necessary with some servers, but if you aren't sure, you should leave it on."`
		Shell         string       `help:"Path to the shell to use to execute actions in. Default looks up bash based on the build.path setting."`
		Platform      []string     `help:"Platform properties to request from remote workers, in the format key=value."`
		CacheDuration cli.Duration `help:"Length of time before we re-check locally cached build actions. Default is unlimited."`
		BuildID       string       `help:"ID of the build action that's being run, to attach to remote requests."`
	} `help:"Settings related to remote execution & caching using the Google remote execution APIs. This section is still experimental and subject to change."`
	Size  map[string]*Size `help:"Named sizes of targets; these are the definitions of what can be passed to the 'size' argument."`
	Cover struct {
		FileExtension    []string `help:"Extensions of files to consider for coverage.\nDefaults to .go, .py, .java, .tsx, .ts, .js, .cc, .h, and .c"`
		ExcludeExtension []string `help:"Extensions of files to exclude from coverage.\nTypically this is for generated code; the default is to exclude protobuf extensions like .pb.go, _pb2.py, etc."`
		ExcludeGlob      []string `help:"Exclude glob patterns from coverage.\nTypically this is for generated code and it is useful when there is no other discrimination possible."`
	} `help:"Configuration relating to coverage reports."`
	Gc struct {
		Keep      []BuildLabel `help:"Marks targets that gc should always keep. Can include meta-targets such as //test/... and //docs:all."`
		KeepLabel []string     `help:"Defines a target label to be kept; for example, if you set this to go, no Go targets would ever be considered for deletion." example:"go"`
	} `help:"Please supports a form of 'garbage collection', by which it means identifying targets that are not used for anything. By default binary targets and all their transitive dependencies are always considered non-garbage, as are any tests directly on those. The config options here allow tweaking this behaviour to retain more things.\n\nNote that it's a very good idea that your BUILD files are in the standard format when running this."`
	Go struct {
		GoTool           string `help:"The binary to use to invoke Go & its subtools with." var:"GO_TOOL"`
		GoRoot           string `help:"If set, will set the GOROOT environment variable appropriately during build actions." var:"GOROOT"`
		GoPath           string `help:"If set, will set the GOPATH environment variable appropriately during build actions." var:"GOPATH"`
		ImportPath       string `help:"Sets the default Go import path at the root of this repository.\nFor example, in the Please repo, we might set it to github.com/thought-machine/please to allow imports from that package within the repo." var:"GO_IMPORT_PATH"`
		CgoCCTool        string `help:"Sets the location of CC while building cgo_library and cgo_test rules. Defaults to gcc" var:"CGO_CC_TOOL"`
		CgoEnabled       string `help:"Sets the CGO_ENABLED which controls whether the cgo build flag is set during cross compilation. Defaults to '0' (disabled)" var:"CGO_ENABLED"`
		FilterTool       string `help:"Sets the location of the please_go_filter tool that is used to filter source files against build constraints." var:"GO_FILTER_TOOL"`
		PleaseGoTool     string `help:"Sets the location of the please_go tool that is used to compile and test go code." var:"PLEASE_GO_TOOL"`
		EmbedTool        string `help:"Sets the location of the please_go_embed tool that is used to parse //go:embed directives." var:"GO_EMBED_TOOL"`
		DelveTool        string `help:"Sets the location of the Delve tool that is used for debugging Go code." var:"DELVE_TOOL"`
		DefaultStatic    bool   `help:"Sets Go binaries to default to static linking. Note that enabling this may have negative consequences for some code, including Go's DNS lookup code in the net module." var:"GO_DEFAULT_STATIC"`
		GoTestRootCompat bool   `help:"Changes the behavior of the build rules to be more compatible with go test i.e. please will descend into the package directory to run unit tests as go test does." var:"GO_TEST_ROOT_COMPAT"`
		CFlags           string `help:"Sets the CFLAGS env var for go rules." var:"GO_C_FLAGS"`
		LDFlags          string `help:"Sets the LDFLAGS env var for go rules." var:"GO_LD_FLAGS"`
	} `help:"Please has built-in support for compiling Go, and of course is written in Go itself.\nSee the config subfields or the Go rules themselves for more information.\n\nNote that Please is a bit more flexible than Go about directory layout - for example, it is possible to have multiple packages in a directory, but it's not a good idea to push this too far since Go's directory layout is inextricably linked with its import paths." exclude_flag:"ExcludeGoRules"`
	Python struct {
		PipTool             string   `help:"The tool that is invoked during pip_library rules." var:"PIP_TOOL"`
		PipFlags            string   `help:"Additional flags to pass to pip invocations in pip_library rules." var:"PIP_FLAGS"`
		PexTool             string   `help:"The tool that's invoked to build pexes. Defaults to please_pex in the install directory." var:"PEX_TOOL"`
		DefaultInterpreter  string   `help:"The interpreter used for python_binary and python_test rules when none is specified on the rule itself. Defaults to python but you could of course set it to, say, pypy." var:"DEFAULT_PYTHON_INTERPRETER"`
		TestRunner          string   `help:"The test runner used to discover & run Python tests; one of unittest, pytest or behave, or a custom import path to bring your own." var:"PYTHON_TEST_RUNNER"`
		TestRunnerBootstrap string   `help:"Target providing test-runner library and its transitive dependencies. Injects plz-provided bootstraps if not given." var:"PYTHON_TEST_RUNNER_BOOTSTRAP"`
		Debugger            string   `help:"Sets what debugger to use to debug Python binaries. The available options are: 'pdb' (default) and 'debugpy'." var:"PYTHON_DEBUGGER"`
		ModuleDir           string   `help:"Defines a directory containing modules from which they can be imported at the top level.\nBy default this is empty but by convention we define our pip_library rules in third_party/python and set this appropriately. Hence any of those third-party libraries that try something like import six will have it work as they expect, even though it's actually in a different location within the .pex." var:"PYTHON_MODULE_DIR"`
		DefaultPipRepo      cli.URL  `help:"Defines a location for a pip repo to download wheels from.\nBy default pip_library uses PyPI (although see below on that) but you may well want to use this define another location to upload your own wheels to.\nIs overridden by the repo argument to pip_library." var:"PYTHON_DEFAULT_PIP_REPO"`
		WheelRepo           cli.URL  `help:"Defines a location for a remote repo that python_wheel rules will download from. See python_wheel for more information." var:"PYTHON_WHEEL_REPO"`
		UsePyPI             bool     `help:"Whether or not to use PyPI for pip_library rules or not. Defaults to true, if you disable this you will presumably want to set DefaultPipRepo to use one of your own.\nIs overridden by the use_pypi argument to pip_library." var:"USE_PYPI"`
		WheelNameScheme     []string `help:"Defines a custom templatized wheel naming scheme. Templatized variables should be surrounded in curly braces, and the available options are: url_base, package_name, version and initial (the first character of package_name). The default search pattern is '{url_base}/{package_name}-{version}-${{OS}}-${{ARCH}}.whl' along with a few common variants." var:"PYTHON_WHEEL_NAME_SCHEME"`
		InterpreterOptions  string   `help:"Options to pass to the python interpeter, when writing shebangs for pex executables." var:"PYTHON_INTERPRETER_OPTIONS"`
		DisableVendorFlags  bool     `help:"Disables injection of vendor specific flags for pip while using pip_library. The option can be useful if you are using something like Pyenv, and the passing of additional flags or configuration that are vendor specific, e.g. --system, breaks your build." var:"DISABLE_VENDOR_FLAGS"`
	} `help:"Please has built-in support for compiling Python.\nPlease's Python artifacts are pex files, which are essentially self-executable zip files containing all needed dependencies, bar the interpreter itself. This fits our aim of at least semi-static binaries for each language.\nSee https://github.com/pantsbuild/pex for more information.\nNote that due to differences between the environment inside a pex and outside some third-party code may not run unmodified (for example, it cannot simply open() files). It's possible to work around a lot of this, but if it all becomes too much it's possible to mark pexes as not zip-safe which typically resolves most of it at a modest speed penalty." exclude_flag:"ExcludePythonRules"`
	Java struct {
		JavacTool          string    `help:"Defines the tool used for the Java compiler. Defaults to javac." var:"JAVAC_TOOL"`
		JlinkTool          string    `help:"Defines the tool used for the Java linker. Defaults to jlink." var:"JLINK_TOOL"`
		JavaHome           string    `help:"Defines the path of the Java Home folder." var:"JAVA_HOME"`
		JavacWorker        string    `help:"Defines the tool used for the Java persistent compiler. This is significantly (approx 4x) faster for large Java trees than invoking javac separately each time. Default to javac_worker in the install directory, but can be switched off to fall back to javactool and separate invocation." var:"JAVAC_WORKER"`
		JarCatTool         string    `help:"Defines the tool used to concatenate .jar files which we use to build the output of java_binary, java_test and various other rules. Defaults to arcat in the internal //_please package." var:"JARCAT_TOOL"`
		JUnitRunner        string    `help:"Defines the .jar containing the JUnit runner. This is built into all java_test rules since it's necessary to make JUnit do anything useful.\nDefaults to junit_runner.jar in the internal //_please package." var:"JUNIT_RUNNER"`
		DefaultTestPackage string    `help:"The Java classpath to search for functions annotated with @Test. If not specified the compiled sources will be searched for files named *Test.java." var:"DEFAULT_TEST_PACKAGE"`
		ReleaseLevel       string    `help:"The default Java release level when compiling.\nSourceLevel and TargetLevel are ignored if this is set. Bear in mind that this flag is only supported in Java version 9+." var:"JAVA_RELEASE_LEVEL"`
		SourceLevel        string    `help:"The default Java source level when compiling. Defaults to 8." var:"JAVA_SOURCE_LEVEL"`
		TargetLevel        string    `help:"The default Java bytecode level to target. Defaults to 8." var:"JAVA_TARGET_LEVEL"`
		JavacFlags         string    `help:"Additional flags to pass to javac when compiling libraries." example:"-Xmx1200M" var:"JAVAC_FLAGS"`
		JavacTestFlags     string    `help:"Additional flags to pass to javac when compiling tests." example:"-Xmx1200M" var:"JAVAC_TEST_FLAGS"`
		DefaultMavenRepo   []cli.URL `help:"Default location to load artifacts from in maven_jar rules. Can be overridden on a per-rule basis." var:"DEFAULT_MAVEN_REPO"`
		Toolchain          string    `help:"A label identifying a java_toolchain." var:"JAVA_TOOLCHAIN"`
	} `help:"Please has built-in support for compiling Java.\nIt builds uber-jars for binary and test rules which contain all dependencies and can be easily deployed, and with the help of some of Please's additional tools they are deterministic as well.\n\nWe've only tested support for Java 7 and 8, although it's likely newer versions will work with little or no change." exclude_flag:"ExcludeJavaRules"`
	Cpp struct {
		CCTool             string     `help:"The tool invoked to compile C code. Defaults to gcc but you might want to set it to clang, for example." var:"CC_TOOL"`
		CppTool            string     `help:"The tool invoked to compile C++ code. Defaults to g++ but you might want to set it to clang++, for example." var:"CPP_TOOL"`
		LdTool             string     `help:"The tool invoked to link object files. Defaults to ld but you could also set it to gold, for example." var:"LD_TOOL"`
		ArTool             string     `help:"The tool invoked to archive static libraries. Defaults to ar." var:"AR_TOOL"`
		LinkWithLdTool     bool       `help:"If true, instructs Please to use the tool set earlier in ldtool to link binaries instead of cctool.\nThis is an esoteric setting that most people don't want; a vanilla ld will not perform all steps necessary here (you'll get lots of missing symbol messages from having no libc etc). Generally best to leave this disabled unless you have very specific requirements." var:"LINK_WITH_LD_TOOL"`
		DefaultOptCflags   string     `help:"Compiler flags passed to all C rules during opt builds; these are typically pretty basic things like what language standard you want to target, warning flags, etc.\nDefaults to --std=c99 -O3 -DNDEBUG -Wall -Wextra -Werror" var:"DEFAULT_OPT_CFLAGS"`
		DefaultDbgCflags   string     `help:"Compiler rules passed to all C rules during dbg builds.\nDefaults to --std=c99 -g3 -DDEBUG -Wall -Wextra -Werror." var:"DEFAULT_DBG_CFLAGS"`
		DefaultOptCppflags string     `help:"Compiler flags passed to all C++ rules during opt builds; these are typically pretty basic things like what language standard you want to target, warning flags, etc.\nDefaults to --std=c++11 -O3 -DNDEBUG -Wall -Wextra -Werror" var:"DEFAULT_OPT_CPPFLAGS"`
		DefaultDbgCppflags string     `help:"Compiler rules passed to all C++ rules during dbg builds.\nDefaults to --std=c++11 -g3 -DDEBUG -Wall -Wextra -Werror." var:"DEFAULT_DBG_CPPFLAGS"`
		DefaultLdflags     string     `help:"Linker flags passed to all C++ rules.\nBy default this is empty." var:"DEFAULT_LDFLAGS"`
		PkgConfigPath      string     `help:"Custom PKG_CONFIG_PATH for pkg-config.\nBy default this is empty." var:"PKG_CONFIG_PATH"`
		Coverage           bool       `help:"If true (the default), coverage will be available for C and C++ build rules.\nThis is still a little experimental but should work for GCC. Right now it does not work for Clang (it likely will in Clang 4.0 which will likely support --fprofile-dir) and so this can be useful to disable it.\nIt's also useful in some cases for CI systems etc if you'd prefer to avoid the overhead, since the tests have to be compiled with extra instrumentation and without optimisation." var:"CPP_COVERAGE"`
		TestMain           BuildLabel `help:"The build target to use for the default main for C++ test rules." example:"///pleasings//cc:unittest_main" var:"CC_TEST_MAIN"`
		ClangModules       bool       `help:"Uses Clang-style arguments for compiling cc_module rules. If disabled gcc-style arguments will be used instead. Experimental, expected to be removed at some point once module compilation methods are more consistent." var:"CC_MODULES_CLANG"`
		DsymTool           string     `help:"Set this to dsymutil or equivalent on macOS to use this tool to generate xcode symbol information for debug builds." var:"DSYM_TOOL"`
	} `help:"Please has built-in support for compiling C and C++ code. We don't support every possible nuance of compilation for these languages, but aim to provide something fairly straightforward.\nTypically there is little problem compiling & linking against system libraries although Please has no insight into those libraries and when they change, so cannot rebuild targets appropriately.\n\nThe C and C++ rules are very similar and simply take a different set of tools and flags to facilitate side-by-side usage." exclude_flag:"ExcludeCCRules"`
	Proto struct {
		ProtocTool       string   `help:"The binary invoked to compile .proto files. Defaults to protoc." var:"PROTOC_TOOL"`
		ProtocGoPlugin   string   `help:"The binary passed to protoc as a plugin to generate Go code. Defaults to protoc-gen-go.\nWe've found this easier to manage with a go_get rule instead though, so you can also pass a build label here. See the Please repo for an example." var:"PROTOC_GO_PLUGIN"`
		GrpcPythonPlugin string   `help:"The plugin invoked to compile Python code for grpc_library.\nDefaults to protoc-gen-grpc-python." var:"GRPC_PYTHON_PLUGIN"`
		GrpcJavaPlugin   string   `help:"The plugin invoked to compile Java code for grpc_library.\nDefaults to protoc-gen-grpc-java." var:"GRPC_JAVA_PLUGIN"`
		GrpcGoPlugin     string   `help:"The plugin invoked to compile Go code for grpc_library.\nIf not set, then the protoc plugin will be used instead." var:"GRPC_GO_PLUGIN"`
		GrpcCCPlugin     string   `help:"The plugin invoked to compile C++ code for grpc_library.\nDefaults to grpc_cpp_plugin." var:"GRPC_CC_PLUGIN"`
		Language         []string `help:"Sets the default set of languages that proto rules are built for.\nChosen from the set of {cc, java, go, py}.\nDefaults to all of them!" var:"PROTO_LANGUAGES"`
		PythonDep        string   `help:"An in-repo dependency that's applied to any Python proto libraries." var:"PROTO_PYTHON_DEP"`
		JavaDep          string   `help:"An in-repo dependency that's applied to any Java proto libraries." var:"PROTO_JAVA_DEP"`
		GoDep            string   `help:"An in-repo dependency that's applied to any Go proto libraries." var:"PROTO_GO_DEP"`
		JsDep            string   `help:"An in-repo dependency that's applied to any Javascript proto libraries." var:"PROTO_JS_DEP"`
		PythonGrpcDep    string   `help:"An in-repo dependency that's applied to any Python gRPC libraries." var:"GRPC_PYTHON_DEP"`
		JavaGrpcDep      string   `help:"An in-repo dependency that's applied to any Java gRPC libraries." var:"GRPC_JAVA_DEP"`
		GoGrpcDep        string   `help:"An in-repo dependency that's applied to any Go gRPC libraries." var:"GRPC_GO_DEP"`
		ProtocFlag       []string `help:"Flags to pass to protoc i.e. the location of well known types. Can be repeated." var:"PROTOC_FLAGS"`
	} `help:"Please has built-in support for compiling protocol buffers, which are a form of codegen to define common data types which can be serialised and communicated between different languages.\nSee https://developers.google.com/protocol-buffers/ for more information.\n\nThere is also support for gRPC, which is an implementation of protobuf's RPC framework. See http://www.grpc.io/ for more information.\n\nNote that you must have the protocol buffers compiler (and gRPC plugins, if needed) installed on your machine to make use of these rules." exclude_flag:"ExcludeProtoRules"`
	Licences struct {
		Accept []string `help:"Licences that are accepted in this repository.\nWhen this is empty licences are ignored. As soon as it's set any licence detected or assigned must be accepted explicitly here.\nThere's no fuzzy matching, so some package managers (especially PyPI and Maven, but shockingly not npm which rather nicely uses SPDX) will generate a lot of slightly different spellings of the same thing, which will all have to be accepted here. We'd rather that than trying to 'cleverly' match them which might result in matching the wrong thing."`
		Reject []string `help:"Licences that are explicitly rejected in this repository.\nAn astute observer will notice that this is not very different to just not adding it to the accept section, but it does have the advantage of explicitly documenting things that the team aren't allowed to use."`
	} `help:"Please has some limited support for declaring acceptable licences and detecting them from some libraries. You should not rely on this for complete licence compliance, but it can be a useful check to try to ensure that unacceptable licences do not slip in."`
	Alias            map[string]*Alias  `help:"Allows defining alias replacements with more detail than the [aliases] section. Otherwise follows the same process, i.e. performs replacements of command strings."`
	Plugin           map[string]*Plugin `help:"Used to define configuration for a Please plugin."`
	PluginDefinition struct {
		Name              string   `help:"The name of the plugin"`
		Description       string   `help:"A description of what the plugin does"`
		BuildDefsDir      []string `help:"Directory to look in when prompted for help topics that aren't known internally. Defaults to build_defs" example:"build_defs"`
		DocumentationSite string   `help:"A link to the documentation for this plugin"`
	} `help:"Set this in your .plzconfig to make the current Please repo a plugin. Add configuration fields with PluginConfig sections"`
	PluginConfig map[string]*PluginConfigDefinition `help:"Defines a new config field for a plugin"`
	Bazel        struct {
		Compatibility bool `help:"Activates limited Bazel compatibility mode. When this is active several rule arguments are available under different names (e.g. compiler_flags -> copts etc), the WORKSPACE file is interpreted, Makefile-style replacements like $< and $@ are made in genrule commands, etc.\nNote that Skylark is not generally supported and many aspects of compatibility are fairly superficial; it's unlikely this will work for complex setups of either tool." var:"BAZEL_COMPATIBILITY"`
	} `help:"Bazel is an open-sourced version of Google's internal build tool. Please draws a lot of inspiration from the original tool although the two have now diverged in various ways.\nNonetheless, if you've used Bazel, you will likely find Please familiar."`

	// buildEnvStored is a cached form of BuildEnv.
	buildEnvStored *storedBuildEnv
	// Profiling can be set to true by a caller to enable CPU profiling in any areas that might
	// want to take special effort about it.
	Profiling bool

	FeatureFlags struct {
		JavaBinaryExecutableByDefault bool `help:"Makes java_binary rules self executable by default. Target release version 16." var:"FF_JAVA_SELF_EXEC"`
		SingleSHA1Hash                bool `help:"Stop combining sha1 with the empty hash when there's a single output (just like SHA256 and the other hash functions do) "`
		PackageOutputsStrictness      bool `help:"Prevents certain combinations of target outputs within a package that result in nondeterminist behaviour"`
		PythonWheelHashing            bool `help:"This hashes the internal build rule that downloads the wheel instead" var:"FF_PYTHON_WHEEL_HASHING"`
		NoIterSourcesMarked           bool `help:"Don't mark sources as done when iterating inputs" var:"FF_NO_ITER_SOURCES_MARKED"`
		ExcludePythonRules            bool `help:"Whether to include the python rules or use the plugin"`
		ExcludeJavaRules              bool `help:"Whether to include the java rules or use the plugin"`
		ExcludeCCRules                bool `help:"Whether to include the C and C++ rules or require use of the plugin"`
		ExcludeGoRules                bool `help:"Whether to include the go rules rules or require use of the plugin"`
		ExcludeShellRules             bool `help:"Whether to include the shell rules rules or require use of the plugin"`
		ExcludeProtoRules             bool `help:"Whether to include the proto rules or require use of the plugin"`
		ExcludeSymlinksInGlob         bool `help:"Whether to include symlinks in the glob" var:"FF_EXCLUDE_GLOB_SYMLINKS"`
		GoDontCollapseImportPath      bool `help:"If set, we will no longer collapse import paths that have repeat final parts e.g. foo/bar/bar -> foo/bar" var:"FF_GO_DONT_COLLAPSE_IMPORT_PATHS"`
	} `help:"Flags controlling preview features for the next release. Typically these config options gate breaking changes and only have a lifetime of one major release."`
	Metrics struct {
		PrometheusGatewayURL string       `help:"The gateway URL to push prometheus updates to."`
		Timeout              cli.Duration `help:"timeout for pushing to the gateway. Defaults to 2 seconds." `
	} `help:"Settings for collecting metrics."`
}

// An Alias represents aliases in the config.
type Alias struct {
	Cmd              string   `help:"Command to run for this alias."`
	Desc             string   `help:"Description of this alias"`
	Subcommand       []string `help:"Known subcommands of this command"`
	Flag             []string `help:"Known flags of this command"`
	PositionalLabels bool     `help:"Treats positional arguments after commands as build labels for the purpose of tab completion."`
}

type Plugin struct {
	Target      BuildLabel          `help:"The build label for the target that provides the plugin repo."`
	ExtraValues map[string][]string `help:"A section of arbitrary key-value properties for the plugin." gcfg:"extra_values"`
}

type PluginConfigDefinition struct {
	ConfigKey    string   `help:"The key of the config field in the .plzconfig file"`
	DefaultValue []string `help:"The default value for this config field, if it has one"`
	Help         string   `help:"The help text to display for this field"`
	Optional     bool     `help:"Whether this config field can be empty"`
	Repeatable   bool     `help:"Whether this config field can be repeated"`
	Inherit      bool     `help:"Whether this config field should be inherited from the host repo or not. Defaults to true."`
	Type         string   `help:"What type to bind this config as e.g. str, bool, or int. Default str."`
}

func (plugin Plugin) copyPlugin() *Plugin {
	values := map[string][]string{}
	for k, v := range plugin.ExtraValues {
		values[k] = v
	}
	plugin.ExtraValues = values
	return &plugin
}

// A Size represents a named size in the config.
type Size struct {
	Timeout     cli.Duration `help:"Timeout for targets of this size"`
	TimeoutName string       `help:"Name of the timeout, to be passed to the 'timeout' argument"`
}

type storedBuildEnv struct {
	Env, Path []string
	Once      sync.Once
}

// Hash returns a hash of the parts of this configuration that affect building targets in general.
// Most parts are considered not to (e.g. cache settings) or affect specific targets (e.g. changing
// tool paths which get accounted for on the targets that use them).
func (config *Configuration) Hash() []byte {
	h := sha1.New()
	// These fields are the ones that need to be in the general hash; other things will be
	// picked up by relevant rules (particularly tool paths etc).
	// Note that container settings are handled separately.
	h.Write([]byte(config.Build.Lang))
	h.Write([]byte(config.Build.Nonce))
	for _, l := range config.Licences.Reject {
		h.Write([]byte(l))
	}
	for _, env := range config.getBuildEnv(false, false) {
		if !strings.HasPrefix(env, "SECRET") {
			h.Write([]byte(env))
		}
	}
	return h.Sum(nil)
}

// GetBuildEnv returns the build environment configured for this config object.
func (config *Configuration) GetBuildEnv() []string {
	config.buildEnvStored.Once.Do(func() {
		config.buildEnvStored.Env = config.getBuildEnv(true, true)
		for _, e := range config.buildEnvStored.Env {
			if strings.HasPrefix(e, "PATH=") {
				config.buildEnvStored.Path = strings.Split(strings.TrimPrefix(e, "PATH="), ":")
			}
		}
	})
	return config.buildEnvStored.Env
}

// EnsurePleaseLocation will resolve `config.Please.Location` to a full path location where it is to be found.
func (config *Configuration) EnsurePleaseLocation() {
	defaultPleaseLocation := fs.ExpandHomePath(DefaultPleaseLocation)

	if config.Please.Location == "" {
		// Determine the location based off where we're running from.
		if exec, err := fs.Executable(); err != nil {
			log.Warning("Can't determine current executable: %s", err)
			config.Please.Location = defaultPleaseLocation
		} else if strings.HasPrefix(exec, defaultPleaseLocation) {
			// Paths within ~/.please are managed by us and have symlinks to subdirectories
			// that we don't want to follow.
			config.Please.Location = defaultPleaseLocation
		} else if deref, err := filepath.EvalSymlinks(exec); err != nil {
			log.Warning("Can't dereference %s: %s", exec, err)
			config.Please.Location = defaultPleaseLocation
		} else {
			config.Please.Location = filepath.Dir(deref)
		}
	} else {
		config.Please.Location = fs.ExpandHomePath(config.Please.Location)
		if !filepath.IsAbs(config.Please.Location) {
			config.Please.Location = filepath.Join(RepoRoot, config.Please.Location)
		}
	}
}

// Path returns the slice of strings corresponding to the PATH env var.
func (config *Configuration) Path() []string {
	config.GetBuildEnv() // ensure it is initialised
	return config.buildEnvStored.Path
}

func (config *Configuration) getBuildEnv(includePath bool, includeUnsafe bool) []string {
	env := []string{}

	// from the BuildEnv config keyword
	for k, v := range config.BuildEnv {
		pair := strings.ReplaceAll(strings.ToUpper(k), "-", "_") + "=" + v
		env = append(env, pair)
	}
	// from the user's environment based on the PassUnsafeEnv config keyword
	if includeUnsafe {
		for _, k := range config.Build.PassUnsafeEnv {
			if v, isSet := os.LookupEnv(k); isSet {
				if k == "PATH" {
					// plz's install location always needs to be on the path.
					v = config.Please.Location + ":" + v
					includePath = false // skip this in a bit
				}
				env = append(env, k+"="+v)
			}
		}
	}
	// from the user's environment based on the PassEnv config keyword
	for _, k := range config.Build.PassEnv {
		if v, isSet := os.LookupEnv(k); isSet {
			if k == "PATH" {
				// plz's install location always needs to be on the path.
				v = config.Please.Location + ":" + v
				includePath = false // skip this in a bit
			}
			env = append(env, k+"="+v)
		}
	}
	if includePath {
		// Use a restricted PATH; it'd be easier for the user if we pass it through
		// but really external environment variables shouldn't affect this.
		// The only concession is that ~ is expanded as the user's home directory
		// in PATH entries.
		env = append(env, "PATH="+strings.Join(append([]string{config.Please.Location}, config.Build.Path...), ":"))
	}

	sort.Strings(env)
	return env
}

// TagsToFields returns a map of string represent the properties of CONFIG object to the config Structfield
func (config *Configuration) TagsToFields() map[string]reflect.StructField {
	tags := make(map[string]reflect.StructField)
	v := reflect.ValueOf(config).Elem()
	for i := 0; i < v.NumField(); i++ {
		if field := v.Field(i); field.Kind() == reflect.Struct {
			for j := 0; j < field.NumField(); j++ {
				if tag := field.Type().Field(j).Tag.Get("var"); tag != "" {
					tags[tag] = field.Type().Field(j)
				}
			}
		}
	}
	return tags
}

// ApplyOverrides applies a set of overrides to the config.
// The keys of the given map are dot notation for the config setting.
func (config *Configuration) ApplyOverrides(overrides map[string]string) error {
	match := func(s1 string) func(string) bool {
		return func(s2 string) bool {
			return strings.ToLower(s2) == s1
		}
	}
	maybeValidOption := func(field reflect.StructField, value, key string) error {
		if options := field.Tag.Get("options"); options != "" {
			if !cli.ContainsString(value, strings.Split(options, ",")) {
				return fmt.Errorf("Invalid value %s for field %s; options are %s", value, key, options)
			}
		}
		return nil
	}
	elem := reflect.ValueOf(config).Elem()
	for k, v := range overrides {
		split := strings.Split(strings.ToLower(k), ".")
		if len(split) == 3 && split[0] == "plugin" {
			if plugin, ok := config.Plugin[split[1]]; ok {
				plugin.ExtraValues[strings.ToLower(split[2])] = []string{v}
				return nil
			}
			log.Fatalf("No plugin with ID %v", split[1])
		}
		if len(split) != 2 {
			return fmt.Errorf("Bad option format: %s", k)
		}

		field := elem.FieldByNameFunc(match(split[0]))
		if !field.IsValid() {
			return fmt.Errorf("Unknown config field: %s", split[0])
		} else if field.Kind() == reflect.Map {
			field.SetMapIndex(reflect.ValueOf(split[1]), reflect.ValueOf(v))
			continue
		} else if field.Kind() != reflect.Struct {
			return fmt.Errorf("Unsettable config field: %s", split[0])
		}
		subfield, ok := field.Type().FieldByNameFunc(match(split[1]))
		if !ok {
			return fmt.Errorf("Unknown config field: %s", split[1])
		}
		field = field.FieldByNameFunc(match(split[1]))
		switch field.Kind() {
		case reflect.String:
			// verify this is a legit setting for this field
			if err := maybeValidOption(subfield, v, k); err != nil {
				return err
			}
			if field.Type().Name() == "URL" {
				field.Set(reflect.ValueOf(cli.URL(v)))
			} else {
				field.Set(reflect.ValueOf(v))
			}
		case reflect.Bool:
			v, _ := gcfgtypes.ParseBool(v)
			field.SetBool(v)
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
			// Comma-separated values are accepted.
			if field.Type().Elem().Kind() == reflect.Struct {
				// Assume it must be a slice of BuildLabel.
				l := []BuildLabel{}
				for _, s := range strings.Split(v, ",") {
					l = append(l, ParseBuildLabel(s, ""))
				}
				field.Set(reflect.ValueOf(l))
			} else if field.Type().Elem().Name() == "URL" {
				urls := []cli.URL{}
				for _, s := range strings.Split(v, ",") {
					urls = append(urls, cli.URL(s))
				}
				field.Set(reflect.ValueOf(urls))
			} else {
				parts := strings.Split(v, ",")
				// verify this is a legit setting for this field
				for _, part := range parts {
					if err := maybeValidOption(subfield, part, k); err != nil {
						return err
					}
				}
				field.Set(reflect.ValueOf(parts))
			}
		default:
			return fmt.Errorf("Can't override config field %s (is %s)", k, field.Kind())
		}
	}

	// Resolve the full path to its location.
	config.EnsurePleaseLocation()

	return nil
}

// Completions returns a list of possible completions for the given option prefix.
func (config *Configuration) Completions(prefix string) []flags.Completion {
	ret := []flags.Completion{}
	t := reflect.TypeOf(config).Elem()
	for i := 0; i < t.NumField(); i++ {
		if field := t.Field(i); field.Type.Kind() == reflect.Struct {
			for j := 0; j < field.Type.NumField(); j++ {
				subfield := field.Type.Field(j)
				if name := strings.ToLower(field.Name + "." + subfield.Name); strings.HasPrefix(name, prefix) {
					help := subfield.Tag.Get("help")
					if options := subfield.Tag.Get("options"); options != "" {
						for _, option := range strings.Split(options, ",") {
							ret = append(ret, flags.Completion{Item: name + ":" + option, Description: help})
						}
					} else {
						ret = append(ret, flags.Completion{Item: name + ":", Description: help})
					}
				}
			}
		}
	}
	return ret
}

// UpdateArgsWithAliases applies the aliases in this config to the given set of arguments.
func (config *Configuration) UpdateArgsWithAliases(args []string) []string {
	for idx, arg := range args[1:] {
		// Please should not touch anything that comes after `--`
		if arg == "--" {
			break
		}
		for k, v := range config.Alias {
			if arg == k {
				// We could insert every token in v into os.Args at this point and then we could have
				// aliases defined in terms of other aliases but that seems rather like overkill so just
				// stick the replacement in wholesale instead.
				// Do not ask about the inner append and the empty slice.
				cmd, err := shlex.Split(v.Cmd)
				if err != nil {
					log.Fatalf("Invalid alias replacement for %s: %s", k, err)
				}
				return append(append(append([]string{}, args[:idx+1]...), cmd...), args[idx+2:]...)
			}
		}
	}
	return args
}

// PrintAliases prints the set of aliases defined in the config.
func (config *Configuration) PrintAliases(w io.Writer) {
	aliases := config.Alias
	names := make([]string, 0, len(aliases))
	maxlen := 0
	for alias := range aliases {
		names = append(names, alias)
		if len(alias) > maxlen {
			maxlen = len(alias)
		}
	}
	sort.Strings(names)
	if len(names) > 0 {
		w.Write([]byte("\nAvailable commands for this repository:\n"))
		tmpl := fmt.Sprintf("  %%-%ds  %%s\n", maxlen)
		for _, name := range names {
			fmt.Fprintf(w, tmpl, name, aliases[name].Desc)
		}
	}
}

// IsABuildFile returns true if given filename is a build file name.
func (config *Configuration) IsABuildFile(name string) bool {
	for _, buildFileName := range config.Parse.BuildFileName {
		if name == buildFileName {
			return true
		}
	}
	return false
}

// NumRemoteExecutors returns the number of actual remote executors we'll have
func (config *Configuration) NumRemoteExecutors() int {
	if config.Remote.URL == "" {
		return 0
	}
	return config.Remote.NumExecutors
}

func (config *Configuration) ShouldLinkGeneratedSources() bool {
	isTruthy, _ := gcfgtypes.ParseBool(config.Build.LinkGeneratedSources)
	return config.Build.LinkGeneratedSources == "hard" || config.Build.LinkGeneratedSources == "soft" || isTruthy
}

func (config Configuration) copyConfig() *Configuration {
	buildConfig := config.BuildConfig
	config.BuildConfig = make(map[string]string, len(buildConfig))
	for k, v := range buildConfig {
		config.BuildConfig[k] = v
	}
	config.buildEnvStored = &storedBuildEnv{}
	plugins := map[string]*Plugin{}
	for name, plugin := range config.Plugin {
		plugins[name] = plugin.copyPlugin()
	}
	config.Plugin = plugins

	pluginConfig := map[string]*PluginConfigDefinition{}
	for key, value := range config.PluginConfig {
		pluginConfig[key] = value
	}

	config.PluginConfig = pluginConfig

	return &config
}

// A ConfigProfile is a string that knows how to handle completions given all the possible config file locations.
type ConfigProfile string

// Complete implements command-line flags completion for a ConfigProfile.
func (profile ConfigProfile) Complete(match string) (completions []flags.Completion) {
	for _, filename := range defaultConfigFiles() {
		matches, _ := filepath.Glob(filename + "." + match + "*")
		for _, match := range matches {
			if suffix := strings.TrimPrefix(match, filename+"."); suffix != "local" { // .plzconfig.local doesn't count
				completions = append(completions, flags.Completion{
					Item:        suffix,
					Description: "Profile defined at " + match,
				})
			}
		}
	}
	return completions
}
